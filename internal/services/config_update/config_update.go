package config_update

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// ==========================================
// Constants & Variables
// ==========================================

// SubscriptionStatus 订阅状态枚举
type SubscriptionStatus int

const (
	StatusNormal          SubscriptionStatus = iota
	StatusExpired                            // 过期
	StatusInactive                           // 失效（被禁用）
	StatusAccountAbnormal                    // 账户异常（被禁用）
	StatusDeviceOverLimit                    // 设备超限
	StatusOldAddress                         // 旧订阅地址
	StatusNotFound                           // 订阅不存在
)

// 预编译正则表达式以提升性能
// 注意：需要匹配完整的链接，包括参数部分（?和#之后的内容）
var nodeLinkPatterns = []*regexp.Regexp{
	// VMess/VLESS: Base64编码的JSON，可能包含参数
	regexp.MustCompile(`(vmess://[^\s]+)`),
	regexp.MustCompile(`(vless://[^\s]+)`),
	// Trojan: UUID@服务器:端口?参数#名称
	regexp.MustCompile(`(trojan://[^\s]+)`),
	// SS: 加密方法:密码@服务器:端口#名称 或 Base64编码格式
	regexp.MustCompile(`(ss://[^\s]+)`),
	// SSR: Base64编码
	regexp.MustCompile(`(ssr://[^\s]+)`),
	// Hysteria: 可能包含参数
	regexp.MustCompile(`(hysteria://[^\s]+)`),
	regexp.MustCompile(`(hysteria2://[^\s]+)`),
	// TUIC: 可能包含参数
	regexp.MustCompile(`(tuic://[^\s]+)`),
	// Naive: 可能包含参数
	regexp.MustCompile(`(naive\+https://[^\s]+)`),
	regexp.MustCompile(`(naive://[^\s]+)`),
	// Anytls: 可能包含参数
	regexp.MustCompile(`(anytls://[^\s]+)`),
}

// Clash 支持的节点类型
var supportedClashTypes = map[string]bool{
	"vmess":     true,
	"vless":     true,
	"trojan":    true,
	"ss":        true,
	"ssr":       true, // Clash Verge/Meta 支持 SSR
	"hysteria":  true,
	"hysteria2": true,
	"tuic":      true,
	"direct":    true, // 信息节点
}

// 地区映射表：支持中英文名称和代码
var regionMap = map[string]string{
	// 亚洲
	"中国": "中国", "CN": "中国", "China": "中国", "中华": "中国", "大陆": "中国", "Mainland": "中国",
	"香港": "香港", "HK": "香港", "Hong Kong": "香港", "HongKong": "香港", "HKG": "香港", "港": "香港",
	"台湾": "台湾", "TW": "台湾", "Taiwan": "台湾", "TWN": "台湾", "台": "台湾",
	"日本": "日本", "JP": "日本", "Japan": "日本", "JPN": "日本", "日": "日本",
	"韩国": "韩国", "KR": "韩国", "Korea": "韩国", "KOR": "韩国", "South Korea": "韩国", "韩": "韩国",
	"新加坡": "新加坡", "SG": "新加坡", "Singapore": "新加坡", "SGP": "新加坡", "狮城": "新加坡",
	"马来西亚": "马来西亚", "MY": "马来西亚", "Malaysia": "马来西亚", "MYS": "马来西亚", "大马": "马来西亚",
	"泰国": "泰国", "TH": "泰国", "Thailand": "泰国", "THA": "泰国", "泰": "泰国",
	"越南": "越南", "VN": "越南", "Vietnam": "越南", "VNM": "越南", "越": "越南",
	"菲律宾": "菲律宾", "PH": "菲律宾", "Philippines": "菲律宾", "PHL": "菲律宾", "菲": "菲律宾",
	"柬埔寨": "柬埔寨", "KH": "柬埔寨", "Cambodia": "柬埔寨", "KHM": "柬埔寨", "柬": "柬埔寨",
	"印度尼西亚": "印度尼西亚", "印尼": "印度尼西亚", "ID": "印度尼西亚", "Indonesia": "印度尼西亚", "IDN": "印度尼西亚",
	"印度": "印度", "IN": "印度", "India": "印度", "IND": "印度", "印": "印度",
	"巴基斯坦": "巴基斯坦", "PK": "巴基斯坦", "Pakistan": "巴基斯坦", "PAK": "巴基斯坦",
	"孟加拉": "孟加拉", "BD": "孟加拉", "Bangladesh": "孟加拉", "BGD": "孟加拉",
	"斯里兰卡": "斯里兰卡", "LK": "斯里兰卡", "Sri Lanka": "斯里兰卡", "LKA": "斯里兰卡",
	"缅甸": "缅甸", "MM": "缅甸", "Myanmar": "缅甸", "MMR": "缅甸",
	"老挝": "老挝", "LA": "老挝", "Laos": "老挝", "LAO": "老挝",
	"文莱": "文莱", "BN": "文莱", "Brunei": "文莱", "BRN": "文莱",
	"蒙古": "蒙古", "MN": "蒙古", "Mongolia": "蒙古", "MNG": "蒙古",
	"哈萨克斯坦": "哈萨克斯坦", "KZ": "哈萨克斯坦", "Kazakhstan": "哈萨克斯坦", "KAZ": "哈萨克斯坦",
	"乌兹别克斯坦": "乌兹别克斯坦", "UZ": "乌兹别克斯坦", "Uzbekistan": "乌兹别克斯坦", "UZB": "乌兹别克斯坦",
	"土耳其": "土耳其", "TR": "土耳其", "Turkey": "土耳其", "TUR": "土耳其", "土": "土耳其",
	"以色列": "以色列", "IL": "以色列", "Israel": "以色列", "ISR": "以色列",
	"阿联酋": "阿联酋", "AE": "阿联酋", "UAE": "阿联酋", "United Arab Emirates": "阿联酋", "迪拜": "阿联酋",
	"沙特阿拉伯": "沙特阿拉伯", "SA": "沙特阿拉伯", "Saudi Arabia": "沙特阿拉伯", "SAU": "沙特阿拉伯",
	"卡塔尔": "卡塔尔", "QA": "卡塔尔", "Qatar": "卡塔尔", "QAT": "卡塔尔",
	"科威特": "科威特", "KW": "科威特", "Kuwait": "科威特", "KWT": "科威特",
	"巴林": "巴林", "BH": "巴林", "Bahrain": "巴林", "BHR": "巴林",
	"阿曼": "阿曼", "OM": "阿曼", "Oman": "阿曼", "OMN": "阿曼",
	"约旦": "约旦", "JO": "约旦", "Jordan": "约旦", "JOR": "约旦",
	"黎巴嫩": "黎巴嫩", "LB": "黎巴嫩", "Lebanon": "黎巴嫩", "LBN": "黎巴嫩",
	"伊拉克": "伊拉克", "IQ": "伊拉克", "Iraq": "伊拉克", "IRQ": "伊拉克",
	"伊朗": "伊朗", "IR": "伊朗", "Iran": "伊朗", "IRN": "伊朗",
	"阿富汗": "阿富汗", "AF": "阿富汗", "Afghanistan": "阿富汗", "AFG": "阿富汗",
	"格鲁吉亚": "格鲁吉亚", "GE": "格鲁吉亚", "Georgia": "格鲁吉亚", "GEO": "格鲁吉亚",
	"亚美尼亚": "亚美尼亚", "AM": "亚美尼亚", "Armenia": "亚美尼亚", "ARM": "亚美尼亚",
	"阿塞拜疆": "阿塞拜疆", "AZ": "阿塞拜疆", "Azerbaijan": "阿塞拜疆", "AZE": "阿塞拜疆",

	// 欧洲
	"英国": "英国", "UK": "英国", "United Kingdom": "英国", "GBR": "英国", "GB": "英国", "England": "英国",
	"德国": "德国", "DE": "德国", "Germany": "德国", "DEU": "德国", "德": "德国",
	"法国": "法国", "FR": "法国", "France": "法国", "FRA": "法国", "法": "法国",
	"意大利": "意大利", "IT": "意大利", "Italy": "意大利", "ITA": "意大利", "意": "意大利",
	"西班牙": "西班牙", "ES": "西班牙", "Spain": "西班牙", "ESP": "西班牙", "西": "西班牙",
	"荷兰": "荷兰", "NL": "荷兰", "Netherlands": "荷兰", "NLD": "荷兰", "Holland": "荷兰", "荷": "荷兰",
	"比利时": "比利时", "BE": "比利时", "Belgium": "比利时", "BEL": "比利时", "比": "比利时",
	"瑞士": "瑞士", "CH": "瑞士", "Switzerland": "瑞士", "CHE": "瑞士",
	"奥地利": "奥地利", "AT": "奥地利", "Austria": "奥地利", "AUT": "奥地利", "奥": "奥地利",
	"瑞典": "瑞典", "SE": "瑞典", "Sweden": "瑞典", "SWE": "瑞典",
	"挪威": "挪威", "NO": "挪威", "Norway": "挪威", "NOR": "挪威", "挪": "挪威",
	"丹麦": "丹麦", "DK": "丹麦", "Denmark": "丹麦", "DNK": "丹麦", "丹": "丹麦",
	"芬兰": "芬兰", "FI": "芬兰", "Finland": "芬兰", "FIN": "芬兰", "芬": "芬兰",
	"波兰": "波兰", "PL": "波兰", "Poland": "波兰", "POL": "波兰", "波": "波兰",
	"捷克": "捷克", "CZ": "捷克", "Czech Republic": "捷克", "CZE": "捷克", "Czech": "捷克",
	"匈牙利": "匈牙利", "HU": "匈牙利", "Hungary": "匈牙利", "HUN": "匈牙利", "匈": "匈牙利",
	"罗马尼亚": "罗马尼亚", "RO": "罗马尼亚", "Romania": "罗马尼亚", "ROU": "罗马尼亚",
	"保加利亚": "保加利亚", "BG": "保加利亚", "Bulgaria": "保加利亚", "BGR": "保加利亚",
	"希腊": "希腊", "GR": "希腊", "Greece": "希腊", "GRC": "希腊", "希": "希腊",
	"葡萄牙": "葡萄牙", "PT": "葡萄牙", "Portugal": "葡萄牙", "PRT": "葡萄牙", "葡": "葡萄牙",
	"爱尔兰": "爱尔兰", "IE": "爱尔兰", "Ireland": "爱尔兰", "IRL": "爱尔兰", "爱": "爱尔兰",
	"冰岛": "冰岛", "IS": "冰岛", "Iceland": "冰岛", "ISL": "冰岛",
	"俄罗斯": "俄罗斯", "RU": "俄罗斯", "Russia": "俄罗斯", "RUS": "俄罗斯", "俄": "俄罗斯",
	"乌克兰": "乌克兰", "UA": "乌克兰", "Ukraine": "乌克兰", "UKR": "乌克兰", "乌": "乌克兰",
	"白俄罗斯": "白俄罗斯", "BY": "白俄罗斯", "Belarus": "白俄罗斯", "BLR": "白俄罗斯",
	"立陶宛": "立陶宛", "LT": "立陶宛", "Lithuania": "立陶宛", "LTU": "立陶宛",
	"拉脱维亚": "拉脱维亚", "LV": "拉脱维亚", "Latvia": "拉脱维亚", "LVA": "拉脱维亚",
	"爱沙尼亚": "爱沙尼亚", "EE": "爱沙尼亚", "Estonia": "爱沙尼亚", "EST": "爱沙尼亚",
	"克罗地亚": "克罗地亚", "HR": "克罗地亚", "Croatia": "克罗地亚", "HRV": "克罗地亚",
	"塞尔维亚": "塞尔维亚", "RS": "塞尔维亚", "Serbia": "塞尔维亚", "SRB": "塞尔维亚",
	"斯洛文尼亚": "斯洛文尼亚", "SI": "斯洛文尼亚", "Slovenia": "斯洛文尼亚", "SVN": "斯洛文尼亚",
	"斯洛伐克": "斯洛伐克", "SK": "斯洛伐克", "Slovakia": "斯洛伐克", "SVK": "斯洛伐克",
	"卢森堡": "卢森堡", "LU": "卢森堡", "Luxembourg": "卢森堡", "LUX": "卢森堡",
	"马耳他": "马耳他", "MT": "马耳他", "Malta": "马耳他", "MLT": "马耳他",
	"塞浦路斯": "塞浦路斯", "CY": "塞浦路斯", "Cyprus": "塞浦路斯", "CYP": "塞浦路斯",

	// 北美洲
	"美国": "美国", "US": "美国", "USA": "美国", "United States": "美国", "United States of America": "美国", "美": "美国",
	"加拿大": "加拿大", "CA": "加拿大", "Canada": "加拿大", "CAN": "加拿大", "加": "加拿大",
	"墨西哥": "墨西哥", "MX": "墨西哥", "Mexico": "墨西哥", "MEX": "墨西哥", "墨": "墨西哥",
	"巴拿马": "巴拿马", "PA": "巴拿马", "Panama": "巴拿马", "PAN": "巴拿马",
	"哥斯达黎加": "哥斯达黎加", "CR": "哥斯达黎加", "Costa Rica": "哥斯达黎加", "CRI": "哥斯达黎加",
	"古巴": "古巴", "CU": "古巴", "Cuba": "古巴", "CUB": "古巴",
	"牙买加": "牙买加", "JM": "牙买加", "Jamaica": "牙买加", "JAM": "牙买加",
	"多米尼加": "多米尼加", "DO": "多米尼加", "Dominican Republic": "多米尼加", "DOM": "多米尼加",
	"危地马拉": "危地马拉", "GT": "危地马拉", "Guatemala": "危地马拉", "GTM": "危地马拉",
	"洪都拉斯": "洪都拉斯", "HN": "洪都拉斯", "Honduras": "洪都拉斯", "HND": "洪都拉斯",
	"萨尔瓦多": "萨尔瓦多", "SV": "萨尔瓦多", "El Salvador": "萨尔瓦多", "SLV": "萨尔瓦多",
	"尼加拉瓜": "尼加拉瓜", "NI": "尼加拉瓜", "Nicaragua": "尼加拉瓜", "NIC": "尼加拉瓜",

	// 南美洲
	"巴西": "巴西", "BR": "巴西", "Brazil": "巴西", "BRA": "巴西", "巴": "巴西",
	"阿根廷": "阿根廷", "AR": "阿根廷", "Argentina": "阿根廷", "ARG": "阿根廷", "阿": "阿根廷",
	"智利": "智利", "CL": "智利", "Chile": "智利", "CHL": "智利", "智": "智利",
	"哥伦比亚": "哥伦比亚", "CO": "哥伦比亚", "Colombia": "哥伦比亚", "COL": "哥伦比亚",
	"秘鲁": "秘鲁", "PE": "秘鲁", "Peru": "秘鲁", "PER": "秘鲁", "秘": "秘鲁",
	"委内瑞拉": "委内瑞拉", "VE": "委内瑞拉", "Venezuela": "委内瑞拉", "VEN": "委内瑞拉",
	"厄瓜多尔": "厄瓜多尔", "EC": "厄瓜多尔", "Ecuador": "厄瓜多尔", "ECU": "厄瓜多尔",
	"玻利维亚": "玻利维亚", "BO": "玻利维亚", "Bolivia": "玻利维亚", "BOL": "玻利维亚",
	"巴拉圭": "巴拉圭", "PY": "巴拉圭", "Paraguay": "巴拉圭", "PRY": "巴拉圭",
	"乌拉圭": "乌拉圭", "UY": "乌拉圭", "Uruguay": "乌拉圭", "URY": "乌拉圭",
	"圭亚那": "圭亚那", "GY": "圭亚那", "Guyana": "圭亚那", "GUY": "圭亚那",
	"苏里南": "苏里南", "SR": "苏里南", "Suriname": "苏里南", "SUR": "苏里南",

	// 大洋洲
	"澳大利亚": "澳大利亚", "澳洲": "澳大利亚", "AU": "澳大利亚", "Australia": "澳大利亚", "AUS": "澳大利亚", "澳": "澳大利亚",
	"新西兰": "新西兰", "NZ": "新西兰", "New Zealand": "新西兰", "NZL": "新西兰", "新": "新西兰",
	"斐济": "斐济", "FJ": "斐济", "Fiji": "斐济", "FJI": "斐济",
	"巴布亚新几内亚": "巴布亚新几内亚", "PG": "巴布亚新几内亚", "Papua New Guinea": "巴布亚新几内亚", "PNG": "巴布亚新几内亚",
	"新喀里多尼亚": "新喀里多尼亚", "NC": "新喀里多尼亚", "New Caledonia": "新喀里多尼亚", "NCL": "新喀里多尼亚",

	// 非洲
	"南非": "南非", "ZA": "南非", "South Africa": "南非", "ZAF": "南非",
	"埃及": "埃及", "EG": "埃及", "Egypt": "埃及", "EGY": "埃及", "埃": "埃及",
	"肯尼亚": "肯尼亚", "KE": "肯尼亚", "Kenya": "肯尼亚", "KEN": "肯尼亚",
	"尼日利亚": "尼日利亚", "NG": "尼日利亚", "Nigeria": "尼日利亚", "NGA": "尼日利亚",
	"摩洛哥": "摩洛哥", "MA": "摩洛哥", "Morocco": "摩洛哥", "MAR": "摩洛哥",
	"阿尔及利亚": "阿尔及利亚", "DZ": "阿尔及利亚", "Algeria": "阿尔及利亚", "DZA": "阿尔及利亚",
	"突尼斯": "突尼斯", "TN": "突尼斯", "Tunisia": "突尼斯", "TUN": "突尼斯",
	"加纳": "加纳", "GH": "加纳", "Ghana": "加纳", "GHA": "加纳",
	"坦桑尼亚": "坦桑尼亚", "TZ": "坦桑尼亚", "Tanzania": "坦桑尼亚", "TZA": "坦桑尼亚",
	"埃塞俄比亚": "埃塞俄比亚", "ET": "埃塞俄比亚", "Ethiopia": "埃塞俄比亚", "ETH": "埃塞俄比亚",
	"乌干达": "乌干达", "UG": "乌干达", "Uganda": "乌干达", "UGA": "乌干达",
	"卢旺达": "卢旺达", "RW": "卢旺达", "Rwanda": "卢旺达", "RWA": "卢旺达",
	"塞内加尔": "塞内加尔", "SN": "塞内加尔", "Senegal": "塞内加尔", "SEN": "塞内加尔",
	"科特迪瓦": "科特迪瓦", "CI": "科特迪瓦", "Ivory Coast": "科特迪瓦", "CIV": "科特迪瓦",
	"喀麦隆": "喀麦隆", "CM": "喀麦隆", "Cameroon": "喀麦隆", "CMR": "喀麦隆",
	"安哥拉": "安哥拉", "AO": "安哥拉", "Angola": "安哥拉", "AGO": "安哥拉",
	"莫桑比克": "莫桑比克", "MZ": "莫桑比克", "Mozambique": "莫桑比克", "MOZ": "莫桑比克",
	"马达加斯加": "马达加斯加", "MG": "马达加斯加", "Madagascar": "马达加斯加", "MDG": "马达加斯加",
	"毛里求斯": "毛里求斯", "MU": "毛里求斯", "Mauritius": "毛里求斯", "MUS": "毛里求斯",
	"塞舌尔": "塞舌尔", "SC": "塞舌尔", "Seychelles": "塞舌尔", "SYC": "塞舌尔",
}

// 服务器域名关键词映射
var serverCodeMap = map[string]string{
	// 亚洲城市
	"tokyo": "日本", "osaka": "日本", "kyoto": "日本", "yokohama": "日本", "nagoya": "日本", "fukuoka": "日本",
	"seoul": "韩国", "busan": "韩国", "incheon": "韩国", "daegu": "韩国",
	"singapore": "新加坡", "sgp": "新加坡",
	"hongkong": "香港", "hong-kong": "香港", "hkg": "香港", "hk": "香港",
	"taipei": "台湾", "taiwan": "台湾", "twn": "台湾",
	"bangkok": "泰国", "thailand": "泰国",
	"kuala-lumpur": "马来西亚", "kualalumpur": "马来西亚", "malaysia": "马来西亚",
	"hanoi": "越南", "ho-chi-minh": "越南", "hochiminh": "越南", "vietnam": "越南",
	"manila": "菲律宾", "philippines": "菲律宾",
	"phnom-penh": "柬埔寨", "phnompenh": "柬埔寨", "cambodia": "柬埔寨",
	"jakarta": "印度尼西亚", "indonesia": "印度尼西亚", "bali": "印度尼西亚",
	"mumbai": "印度", "delhi": "印度", "bangalore": "印度", "chennai": "印度", "kolkata": "印度", "india": "印度",
	"karachi": "巴基斯坦", "lahore": "巴基斯坦", "islamabad": "巴基斯坦", "pakistan": "巴基斯坦",
	"dhaka": "孟加拉", "bangladesh": "孟加拉",
	"colombo": "斯里兰卡", "sri-lanka": "斯里兰卡", "srilanka": "斯里兰卡",
	"yangon": "缅甸", "myanmar": "缅甸",
	"vientiane": "老挝", "laos": "老挝",
	"bandar-seri-begawan": "文莱", "brunei": "文莱",
	"ulaanbaatar": "蒙古", "mongolia": "蒙古",
	"almaty": "哈萨克斯坦", "astana": "哈萨克斯坦", "kazakhstan": "哈萨克斯坦",
	"tashkent": "乌兹别克斯坦", "uzbekistan": "乌兹别克斯坦",
	"istanbul": "土耳其", "ankara": "土耳其", "turkey": "土耳其",
	"tel-aviv": "以色列", "telaviv": "以色列", "jerusalem": "以色列", "israel": "以色列",
	"dubai": "阿联酋", "abu-dhabi": "阿联酋", "abudhabi": "阿联酋", "uae": "阿联酋",
	"riyadh": "沙特阿拉伯", "jeddah": "沙特阿拉伯", "saudi": "沙特阿拉伯",
	"doha": "卡塔尔", "qatar": "卡塔尔",
	"kuwait": "科威特", "kuwait-city": "科威特",
	"manama": "巴林", "bahrain": "巴林",
	"muscat": "阿曼", "oman": "阿曼",
	"amman": "约旦", "jordan": "约旦",
	"beirut": "黎巴嫩", "lebanon": "黎巴嫩",
	"baghdad": "伊拉克", "iraq": "伊拉克",
	"tehran": "伊朗", "iran": "伊朗",
	"kabul": "阿富汗", "afghanistan": "阿富汗",
	"tbilisi": "格鲁吉亚", "georgia": "格鲁吉亚",
	"yerevan": "亚美尼亚", "armenia": "亚美尼亚",
	"baku": "阿塞拜疆", "azerbaijan": "阿塞拜疆",

	// 欧洲城市
	"london": "英国", "manchester": "英国", "birmingham": "英国", "edinburgh": "英国", "uk": "英国", "england": "英国",
	"berlin": "德国", "munich": "德国", "frankfurt": "德国", "hamburg": "德国", "cologne": "德国", "germany": "德国",
	"paris": "法国", "lyon": "法国", "marseille": "法国", "toulouse": "法国", "france": "法国",
	"rome": "意大利", "milan": "意大利", "naples": "意大利", "turin": "意大利", "italy": "意大利",
	"madrid": "西班牙", "barcelona": "西班牙", "valencia": "西班牙", "seville": "西班牙", "spain": "西班牙",
	"amsterdam": "荷兰", "rotterdam": "荷兰", "the-hague": "荷兰", "hague": "荷兰", "netherlands": "荷兰", "holland": "荷兰",
	"brussels": "比利时", "antwerp": "比利时", "ghent": "比利时", "belgium": "比利时",
	"zurich": "瑞士", "geneva": "瑞士", "basel": "瑞士", "bern": "瑞士", "switzerland": "瑞士",
	"vienna": "奥地利", "salzburg": "奥地利", "graz": "奥地利", "austria": "奥地利",
	"stockholm": "瑞典", "gothenburg": "瑞典", "malmo": "瑞典", "sweden": "瑞典",
	"oslo": "挪威", "bergen": "挪威", "trondheim": "挪威", "norway": "挪威",
	"copenhagen": "丹麦", "aarhus": "丹麦", "odense": "丹麦", "denmark": "丹麦",
	"helsinki": "芬兰", "tampere": "芬兰", "turku": "芬兰", "finland": "芬兰",
	"warsaw": "波兰", "krakow": "波兰", "gdansk": "波兰", "poland": "波兰",
	"prague": "捷克", "brno": "捷克", "czech": "捷克", "czech-republic": "捷克",
	"budapest": "匈牙利", "debrecen": "匈牙利", "hungary": "匈牙利",
	"bucharest": "罗马尼亚", "cluj-napoca": "罗马尼亚", "romania": "罗马尼亚",
	"sofia": "保加利亚", "plovdiv": "保加利亚", "bulgaria": "保加利亚",
	"athens": "希腊", "thessaloniki": "希腊", "greece": "希腊",
	"lisbon": "葡萄牙", "porto": "葡萄牙", "portugal": "葡萄牙",
	"dublin": "爱尔兰", "cork": "爱尔兰", "ireland": "爱尔兰",
	"reykjavik": "冰岛", "iceland": "冰岛",
	"moscow": "俄罗斯", "saint-petersburg": "俄罗斯", "st-petersburg": "俄罗斯", "novosibirsk": "俄罗斯", "russia": "俄罗斯",
	"kyiv": "乌克兰", "kiev": "乌克兰", "kharkiv": "乌克兰", "odessa": "乌克兰", "ukraine": "乌克兰",
	"minsk": "白俄罗斯", "belarus": "白俄罗斯",
	"vilnius": "立陶宛", "kaunas": "立陶宛", "lithuania": "立陶宛",
	"riga": "拉脱维亚", "latvia": "拉脱维亚",
	"tallinn": "爱沙尼亚", "estonia": "爱沙尼亚",
	"zagreb": "克罗地亚", "split": "克罗地亚", "croatia": "克罗地亚",
	"belgrade": "塞尔维亚", "novi-sad": "塞尔维亚", "serbia": "塞尔维亚",
	"ljubljana": "斯洛文尼亚", "maribor": "斯洛文尼亚", "slovenia": "斯洛文尼亚",
	"bratislava": "斯洛伐克", "kosice": "斯洛伐克", "slovakia": "斯洛伐克",
	"luxembourg": "卢森堡", "luxemburg": "卢森堡",
	"valletta": "马耳他", "malta": "马耳他",
	"nicosia": "塞浦路斯", "cyprus": "塞浦路斯",

	// 北美洲城市
	"new-york": "美国", "newyork": "美国", "nyc": "美国", "los-angeles": "美国", "losangeles": "美国", "la": "美国",
	"chicago": "美国", "houston": "美国", "phoenix": "美国", "philadelphia": "美国", "san-antonio": "美国",
	"san-diego": "美国", "dallas": "美国", "san-jose": "美国", "austin": "美国", "jacksonville": "美国",
	"san-francisco": "美国", "sanfrancisco": "美国", "sf": "美国", "seattle": "美国", "miami": "美国",
	"boston": "美国", "denver": "美国", "atlanta": "美国", "detroit": "美国", "portland": "美国",
	"america": "美国", "usa": "美国", "united-states": "美国",
	"toronto": "加拿大", "vancouver": "加拿大", "montreal": "加拿大", "calgary": "加拿大", "ottawa": "加拿大",
	"edmonton": "加拿大", "winnipeg": "加拿大", "quebec": "加拿大", "canada": "加拿大",
	"mexico-city": "墨西哥", "mexicocity": "墨西哥", "guadalajara": "墨西哥", "monterrey": "墨西哥", "mexico": "墨西哥",
	"panama-city": "巴拿马", "panamacity": "巴拿马", "panama": "巴拿马",
	"san-jose-cr": "哥斯达黎加", "costarica": "哥斯达黎加", "costa-rica": "哥斯达黎加",
	"havana": "古巴", "cuba": "古巴",
	"kingston": "牙买加", "jamaica": "牙买加",
	"santo-domingo": "多米尼加", "dominican": "多米尼加",
	"guatemala-city": "危地马拉", "guatemala": "危地马拉",
	"tegucigalpa": "洪都拉斯", "honduras": "洪都拉斯",
	"san-salvador": "萨尔瓦多", "elsalvador": "萨尔瓦多",
	"managua": "尼加拉瓜", "nicaragua": "尼加拉瓜",

	// 南美洲城市
	"sao-paulo": "巴西", "saopaulo": "巴西", "rio-de-janeiro": "巴西", "riodejaneiro": "巴西", "brasilia": "巴西",
	"belo-horizonte": "巴西", "salvador": "巴西", "fortaleza": "巴西", "curitiba": "巴西", "brazil": "巴西",
	"buenos-aires": "阿根廷", "buenosaires": "阿根廷", "cordoba": "阿根廷", "rosario": "阿根廷", "argentina": "阿根廷",
	"santiago": "智利", "valparaiso": "智利", "concepcion": "智利", "chile": "智利",
	"bogota": "哥伦比亚", "medellin": "哥伦比亚", "cali": "哥伦比亚", "colombia": "哥伦比亚",
	"lima": "秘鲁", "arequipa": "秘鲁", "trujillo": "秘鲁", "peru": "秘鲁",
	"caracas": "委内瑞拉", "maracaibo": "委内瑞拉", "valencia-ve": "委内瑞拉", "venezuela": "委内瑞拉",
	"quito": "厄瓜多尔", "guayaquil": "厄瓜多尔", "ecuador": "厄瓜多尔",
	"la-paz": "玻利维亚", "lapaz": "玻利维亚", "santa-cruz": "玻利维亚", "bolivia": "玻利维亚",
	"asuncion": "巴拉圭", "paraguay": "巴拉圭",
	"montevideo": "乌拉圭", "uruguay": "乌拉圭",
	"georgetown": "圭亚那", "guyana": "圭亚那",
	"paramaribo": "苏里南", "suriname": "苏里南",

	// 大洋洲城市
	"sydney": "澳大利亚", "melbourne": "澳大利亚", "brisbane": "澳大利亚", "perth": "澳大利亚", "adelaide": "澳大利亚",
	"canberra": "澳大利亚", "darwin": "澳大利亚", "hobart": "澳大利亚", "australia": "澳大利亚",
	"auckland": "新西兰", "wellington": "新西兰", "christchurch": "新西兰", "newzealand": "新西兰", "nz": "新西兰",
	"suva": "斐济", "fiji": "斐济",
	"port-moresby": "巴布亚新几内亚", "papua": "巴布亚新几内亚",
	"noumea": "新喀里多尼亚", "new-caledonia": "新喀里多尼亚",

	// 非洲城市
	"johannesburg": "南非", "cape-town": "南非", "capetown": "南非", "durban": "南非", "southafrica": "南非",
	"cairo": "埃及", "alexandria": "埃及", "giza": "埃及", "egypt": "埃及",
	"nairobi": "肯尼亚", "mombasa": "肯尼亚", "kenya": "肯尼亚",
	"lagos": "尼日利亚", "abuja": "尼日利亚", "kano": "尼日利亚", "nigeria": "尼日利亚",
	"casablanca": "摩洛哥", "rabat": "摩洛哥", "marrakech": "摩洛哥", "morocco": "摩洛哥",
	"algiers": "阿尔及利亚", "orán": "阿尔及利亚", "algeria": "阿尔及利亚",
	"tunis": "突尼斯", "sfax": "突尼斯", "tunisia": "突尼斯",
	"accra": "加纳", "kumasi": "加纳", "ghana": "加纳",
	"dar-es-salaam": "坦桑尼亚", "dodoma": "坦桑尼亚", "tanzania": "坦桑尼亚",
	"addis-ababa": "埃塞俄比亚", "ethiopia": "埃塞俄比亚",
	"kampala": "乌干达", "uganda": "乌干达",
	"kigali": "卢旺达", "rwanda": "卢旺达",
	"dakar": "塞内加尔", "senegal": "塞内加尔",
	"abidjan": "科特迪瓦", "ivory-coast": "科特迪瓦",
	"douala": "喀麦隆", "yaounde": "喀麦隆", "cameroon": "喀麦隆",
	"luanda": "安哥拉", "angola": "安哥拉",
	"maputo": "莫桑比克", "mozambique": "莫桑比克",
	"antananarivo": "马达加斯加", "madagascar": "马达加斯加",
	"port-louis": "毛里求斯", "mauritius": "毛里求斯",
	"victoria": "塞舌尔", "seychelles": "塞舌尔",
}

// 地区关键词（按长度排序）
var regionKeys = []string{
	// 长关键词优先
	"巴布亚新几内亚", "新喀里多尼亚", "阿拉伯联合酋长国", "沙特阿拉伯", "乌兹别克斯坦", "哈萨克斯坦", "印度尼西亚",
	"美利坚合众国", "United States of America", "United Arab Emirates", "Papua New Guinea", "New Caledonia",
	"South Korea", "New Zealand", "South Africa", "Ivory Coast", "El Salvador", "Costa Rica", "Dominican Republic",
	// 中等长度关键词
	"香港", "台湾", "日本", "韩国", "新加坡", "美国", "英国", "德国", "法国", "加拿大", "澳大利亚", "新西兰",
	"印度", "俄罗斯", "荷兰", "泰国", "马来西亚", "越南", "菲律宾", "柬埔寨", "印度尼西亚", "巴基斯坦", "孟加拉",
	"斯里兰卡", "缅甸", "老挝", "文莱", "蒙古", "土耳其", "以色列", "阿联酋", "沙特", "卡塔尔", "科威特", "巴林",
	"阿曼", "约旦", "黎巴嫩", "伊拉克", "伊朗", "阿富汗", "格鲁吉亚", "亚美尼亚", "阿塞拜疆",
	"意大利", "西班牙", "比利时", "瑞士", "奥地利", "瑞典", "挪威", "丹麦", "芬兰", "波兰", "捷克", "匈牙利",
	"罗马尼亚", "保加利亚", "希腊", "葡萄牙", "爱尔兰", "冰岛", "乌克兰", "白俄罗斯", "立陶宛", "拉脱维亚",
	"爱沙尼亚", "克罗地亚", "塞尔维亚", "斯洛文尼亚", "斯洛伐克", "卢森堡", "马耳他", "塞浦路斯",
	"墨西哥", "巴拿马", "哥斯达黎加", "古巴", "牙买加", "多米尼加", "危地马拉", "洪都拉斯", "萨尔瓦多", "尼加拉瓜",
	"巴西", "阿根廷", "智利", "哥伦比亚", "秘鲁", "委内瑞拉", "厄瓜多尔", "玻利维亚", "巴拉圭", "乌拉圭", "圭亚那", "苏里南",
	"斐济", "南非", "埃及", "肯尼亚", "尼日利亚", "摩洛哥", "阿尔及利亚", "突尼斯", "加纳", "坦桑尼亚",
	"埃塞俄比亚", "乌干达", "卢旺达", "塞内加尔", "科特迪瓦", "喀麦隆", "安哥拉", "莫桑比克", "马达加斯加", "毛里求斯", "塞舌尔",
	// 英文全称
	"China", "Hong Kong", "Taiwan", "Japan", "Korea", "Singapore", "USA", "UK", "Germany", "France", "Canada",
	"Australia", "India", "Russia", "Netherlands", "Thailand", "Malaysia", "Vietnam", "Philippines", "Cambodia",
	"Indonesia", "Pakistan", "Bangladesh", "Sri Lanka", "Myanmar", "Laos", "Brunei", "Mongolia", "Kazakhstan",
	"Uzbekistan", "Turkey", "Israel", "Qatar", "Kuwait", "Bahrain", "Oman", "Jordan", "Lebanon", "Iraq", "Iran",
	"Afghanistan", "Georgia", "Armenia", "Azerbaijan", "Italy", "Spain", "Belgium", "Switzerland", "Austria",
	"Sweden", "Norway", "Denmark", "Finland", "Poland", "Czech", "Hungary", "Romania", "Bulgaria", "Greece",
	"Portugal", "Ireland", "Iceland", "Ukraine", "Belarus", "Lithuania", "Latvia", "Estonia", "Croatia", "Serbia",
	"Slovenia", "Slovakia", "Luxembourg", "Malta", "Cyprus", "Mexico", "Panama", "Cuba", "Jamaica", "Guatemala",
	"Honduras", "Nicaragua", "Brazil", "Argentina", "Chile", "Colombia", "Peru", "Venezuela", "Ecuador", "Bolivia",
	"Paraguay", "Uruguay", "Guyana", "Suriname", "Fiji", "South Africa", "Egypt", "Kenya", "Nigeria", "Morocco",
	"Algeria", "Tunisia", "Ghana", "Tanzania", "Ethiopia", "Uganda", "Rwanda", "Senegal", "Cameroon", "Angola",
	"Mozambique", "Madagascar", "Mauritius", "Seychelles",
	// 国家代码（2-3字母）
	"CN", "HK", "TW", "JP", "KR", "SG", "US", "UK", "GB", "DE", "FR", "CA", "AU", "NZ", "IN", "RU", "NL", "TH",
	"MY", "VN", "PH", "KH", "ID", "PK", "BD", "LK", "MM", "LA", "BN", "MN", "KZ", "UZ", "TR", "IL", "AE", "SA",
	"QA", "KW", "BH", "OM", "JO", "LB", "IQ", "IR", "AF", "GE", "AM", "AZ", "IT", "ES", "BE", "CH", "AT", "SE",
	"NO", "DK", "FI", "PL", "CZ", "HU", "RO", "BG", "GR", "PT", "IE", "IS", "UA", "BY", "LT", "LV", "EE", "HR",
	"RS", "SI", "SK", "LU", "MT", "CY", "MX", "PA", "CR", "CU", "JM", "DO", "GT", "HN", "SV", "NI", "BR", "AR",
	"CL", "CO", "PE", "VE", "EC", "BO", "PY", "UY", "GY", "SR", "FJ", "PG", "NC", "ZA", "EG", "KE", "NG", "MA",
	"DZ", "TN", "GH", "TZ", "ET", "UG", "RW", "SN", "CI", "CM", "AO", "MZ", "MG", "MU", "SC",
	// 简写
	"港", "台", "日", "韩", "新", "美", "英", "德", "法", "加", "澳", "印", "俄", "荷", "泰", "马", "越", "菲", "柬",
	"土", "埃", "巴", "智", "秘", "阿",
}

// ==========================================
// Types
// ==========================================

// SubscriptionContext 订阅上下文，包含生成配置所需的所有信息
type SubscriptionContext struct {
	User           models.User
	Subscription   models.Subscription
	Proxies        []*ProxyNode
	Status         SubscriptionStatus
	ResetRecord    *models.SubscriptionReset // 如果是旧订阅地址，这里会有记录
	CurrentDevices int
	DeviceLimit    int
}

// ConfigUpdateService 配置更新服务
type ConfigUpdateService struct {
	db           *gorm.DB
	isRunning    bool
	runningMutex sync.Mutex
	// 缓存站点URL，避免频繁查询
	siteURL string
	// 缓存客服QQ
	supportQQ string
}

// nodeWithOrder 用于排序导入
type nodeWithOrder struct {
	node       *ProxyNode
	orderIndex int
}

// ==========================================
// Service Lifecycle
// ==========================================

// NewConfigUpdateService 创建配置更新服务
func NewConfigUpdateService() *ConfigUpdateService {
	service := &ConfigUpdateService{
		db: database.GetDB(),
	}
	// 初始化缓存配置
	service.refreshSystemConfig()
	return service
}

// refreshSystemConfig 刷新系统配置缓存
func (s *ConfigUpdateService) refreshSystemConfig() {
	// 获取网站域名（使用公共函数）
	domain := utils.GetDomainFromDB(s.db)
	if domain != "" {
		s.siteURL = utils.FormatDomainURL(domain)
	} else {
		s.siteURL = "请在系统设置中配置域名"
	}

	// 获取客服QQ（只从 category = "general" 获取）
	var supportQQConfig models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "support_qq", "general").First(&supportQQConfig).Error; err == nil && supportQQConfig.Value != "" {
		s.supportQQ = strings.TrimSpace(supportQQConfig.Value)
	} else {
		s.supportQQ = "" // 不设置默认值，如果未配置则为空
	}
}

// ==========================================
// Public API
// ==========================================

// IsRunning 检查是否正在运行
func (s *ConfigUpdateService) IsRunning() bool {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()
	return s.isRunning
}

// GetStatus 获取状态
func (s *ConfigUpdateService) GetStatus() map[string]interface{} {
	var lastUpdate string
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error; err == nil {
		lastUpdate = config.Value
	}

	return map[string]interface{}{
		"is_running":  s.IsRunning(),
		"last_update": lastUpdate,
		"next_update": "",
	}
}

// GetLogs 获取日志
func (s *ConfigUpdateService) GetLogs(limit int) []map[string]interface{} {
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_logs").First(&config).Error; err != nil {
		return []map[string]interface{}{}
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal([]byte(config.Value), &logs); err != nil {
		return []map[string]interface{}{}
	}

	if len(logs) > limit {
		return logs[len(logs)-limit:]
	}
	return logs
}

// ClearLogs 清理日志
func (s *ConfigUpdateService) ClearLogs() error {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_logs").First(&config).Error

	if err != nil {
		config = models.SystemConfig{
			Key:         "config_update_logs",
			Value:       "[]",
			Type:        "json",
			Category:    "general",
			DisplayName: "配置更新日志",
			Description: "配置更新任务日志",
		}
		return s.db.Create(&config).Error
	}
	config.Value = "[]"
	return s.db.Save(&config).Error
}

// GetConfig 获取配置（公开方法）
func (s *ConfigUpdateService) GetConfig() (map[string]interface{}, error) {
	return s.getConfig()
}

// RunUpdateTask 执行配置更新任务
func (s *ConfigUpdateService) RunUpdateTask() error {
	s.runningMutex.Lock()
	if s.isRunning {
		s.runningMutex.Unlock()
		return fmt.Errorf("任务已在运行中")
	}
	s.isRunning = true
	s.runningMutex.Unlock()

	defer func() {
		s.runningMutex.Lock()
		s.isRunning = false
		s.runningMutex.Unlock()
	}()

	s.log("INFO", "开始执行配置更新任务")

	// 获取配置
	config, err := s.getConfig()
	if err != nil {
		s.log("ERROR", fmt.Sprintf("获取配置失败: %v", err))
		return err
	}

	urls := config["urls"].([]string)
	if len(urls) == 0 {
		msg := "未配置节点源URL"
		s.log("ERROR", msg)
		return fmt.Errorf(msg)
	}

	s.log("INFO", fmt.Sprintf("获取到 %d 个节点源URL", len(urls)))

	// 1. 获取节点
	nodes, err := s.FetchNodesFromURLs(urls)
	if err != nil {
		s.log("ERROR", fmt.Sprintf("获取节点失败: %v", err))
		return err
	}

	if len(nodes) == 0 {
		msg := "未获取到有效节点"
		s.log("WARN", msg)
		return fmt.Errorf(msg)
	}

	s.log("INFO", fmt.Sprintf("共获取到 %d 个有效节点链接，准备入库", len(nodes)))

	// 2. 解析节点并整理准备入库
	nodesWithOrder, stats := s.processFetchedNodes(urls, nodes)

	// 输出统计信息
	if stats.parseFailed > 0 {
		s.log("WARN", fmt.Sprintf("解析失败的节点: %d 个", stats.parseFailed))
	}
	if stats.duplicates > 0 {
		s.log("INFO", fmt.Sprintf("去重跳过的节点: %d 个", stats.duplicates))
	}
	if stats.invalidLinks > 0 {
		s.log("WARN", fmt.Sprintf("无效链接的节点: %d 个", stats.invalidLinks))
	}
	s.log("INFO", fmt.Sprintf("成功解析并准备入库的节点: %d 个", len(nodesWithOrder)))

	// 3. 入库
	importedCount := s.importNodesToDatabaseWithOrder(nodesWithOrder)
	s.updateLastUpdateTime()

	s.log("SUCCESS", fmt.Sprintf("任务完成: 解析出 %d 个节点，成功入库/更新 %d 个", len(nodesWithOrder), importedCount))
	return nil
}

// ==========================================
// Internal Logic
// ==========================================

// updateStats 统计信息结构
type updateStats struct {
	parseFailed   int
	duplicates    int
	invalidLinks  int
	missingSource int
}

// processFetchedNodes 处理获取到的节点：分组、去重、排序
func (s *ConfigUpdateService) processFetchedNodes(urls []string, nodes []map[string]interface{}) ([]nodeWithOrder, updateStats) {
	var nodesWithOrder []nodeWithOrder
	stats := updateStats{}

	seenKeys := make(map[string]bool)
	usedNames := make(map[string]bool)

	// 按订阅地址分组节点
	nodesByURL := make(map[string][]map[string]interface{})
	for _, nodeInfo := range nodes {
		sourceURL, _ := nodeInfo["source_url"].(string)
		if sourceURL == "" {
			stats.missingSource++
			s.log("WARN", fmt.Sprintf("节点缺少来源URL，跳过: %v", nodeInfo))
			continue
		}
		nodesByURL[sourceURL] = append(nodesByURL[sourceURL], nodeInfo)
	}

	if stats.missingSource > 0 {
		s.log("WARN", fmt.Sprintf("共 %d 个节点缺少来源URL", stats.missingSource))
	}

	// 按照订阅地址的顺序处理节点
	for urlIndex, url := range urls {
		urlNodes := nodesByURL[url]
		nodeIndexInURL := 0

		if len(urlNodes) > 0 {
			s.log("INFO", fmt.Sprintf("开始处理订阅地址 [%d/%d] 的节点，共 %d 个链接", urlIndex+1, len(urls), len(urlNodes)))
		}

		for _, nodeInfo := range urlNodes {
			link, ok := nodeInfo["url"].(string)
			if !ok {
				stats.invalidLinks++
				s.log("WARN", fmt.Sprintf("节点链接格式无效，跳过: %v", nodeInfo))
				continue
			}

			node, err := ParseNodeLink(link)
			if err != nil {
				stats.parseFailed++
				linkPreview := link
				if len(linkPreview) > 50 {
					linkPreview = linkPreview[:50] + "..."
				}
				s.log("WARN", fmt.Sprintf("解析节点失败: %v, 链接: %s", err, linkPreview))
				continue
			}

			// 生成去重键（统一使用 : 分隔符）
			key := s.generateNodeDedupKey(node.Type, node.Server, node.Port)
			if seenKeys[key] {
				stats.duplicates++
				s.log("DEBUG", fmt.Sprintf("节点重复，跳过: %s (%s:%s:%d)", node.Name, node.Type, node.Server, node.Port))
				continue
			}
			seenKeys[key] = true

			// 处理名称重复
			originalName := node.Name
			newName := originalName
			counter := 1
			for usedNames[newName] {
				newName = fmt.Sprintf("%s-%d", originalName, counter)
				counter++
			}
			if newName != originalName {
				s.log("DEBUG", fmt.Sprintf("节点名称重复，重命名为: %s -> %s", originalName, newName))
			}
			node.Name = newName
			usedNames[newName] = true

			// 计算顺序索引：source_url_index * 10000 + node_index_in_url
			orderIndex := urlIndex*10000 + nodeIndexInURL
			nodeIndexInURL++

			nodesWithOrder = append(nodesWithOrder, nodeWithOrder{
				node:       node,
				orderIndex: orderIndex,
			})
		}
	}
	return nodesWithOrder, stats
}

// getConfig 获取配置
func (s *ConfigUpdateService) getConfig() (map[string]interface{}, error) {
	var configs []models.SystemConfig
	s.db.Where("category = ?", "config_update").Find(&configs)

	result := map[string]interface{}{
		"urls":              []string{},
		"target_dir":        "./uploads/config",
		"v2ray_file":        "xr",
		"clash_file":        "clash.yaml",
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	var urlsConfig *models.SystemConfig

	for _, config := range configs {
		if config.Key == "urls" {
			urlsConfig = &config
		} else {
			result[config.Key] = config.Value
		}
	}

	// 处理 URLs
	if urlsConfig != nil && strings.TrimSpace(urlsConfig.Value) != "" {
		var filtered []string
		for _, u := range strings.Split(urlsConfig.Value, "\n") {
			if u = strings.TrimSpace(u); u != "" {
				filtered = append(filtered, u)
			}
		}
		result["urls"] = filtered
	}

	return result, nil
}

// updateLastUpdateTime 更新最后更新时间
func (s *ConfigUpdateService) updateLastUpdateTime() {
	now := utils.GetBeijingTime().Format("2006-01-02T15:04:05")
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error

	if err != nil {
		config = models.SystemConfig{
			Key:         "config_update_last_update",
			Value:       now,
			Type:        "string",
			Category:    "config_update",
			DisplayName: "最后更新时间",
			Description: "配置更新任务的最后执行时间",
		}
		s.db.Create(&config)
	} else {
		config.Value = now
		s.db.Save(&config)
	}
}

// log 记录日志
func (s *ConfigUpdateService) log(level, message string) {
	now := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
	logEntry := map[string]interface{}{
		"time":    now,
		"level":   level,
		"message": message,
	}

	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_logs").First(&config).Error; err != nil {
		initialLogs := []map[string]interface{}{logEntry}
		logsJSON, _ := json.Marshal(initialLogs)
		config = models.SystemConfig{
			Key:         "config_update_logs",
			Value:       string(logsJSON),
			Type:        "json",
			Category:    "config_update",
			DisplayName: "配置更新日志",
			Description: "配置更新任务日志",
		}
		s.db.Create(&config)
	} else {
		var logs []map[string]interface{}
		json.Unmarshal([]byte(config.Value), &logs)
		logs = append(logs, logEntry)

		// 限制日志数量，保留最近 100 条
		if len(logs) > 100 {
			logs = logs[len(logs)-100:]
		}

		logsJSON, _ := json.Marshal(logs)
		config.Value = string(logsJSON)
		s.db.Save(&config)
	}

	// 同时打印到系统日志
	if utils.AppLogger != nil {
		if level == "ERROR" {
			utils.AppLogger.Error(message)
		} else {
			utils.AppLogger.Info(message)
		}
	}
}

// ==========================================
// Node Processing
// ==========================================

// FetchNodesFromURLs 从URL列表获取节点
func (s *ConfigUpdateService) FetchNodesFromURLs(urls []string) ([]map[string]interface{}, error) {
	var allNodes []map[string]interface{}

	for i, url := range urls {
		s.log("INFO", fmt.Sprintf("正在下载节点源 [%d/%d]: %s", i+1, len(urls), url))

		resp, err := http.Get(url)
		if err != nil {
			s.log("ERROR", fmt.Sprintf("下载失败: %v", err))
			continue
		}
		defer resp.Body.Close()

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			s.log("ERROR", fmt.Sprintf("读取内容失败: %v", err))
			continue
		}

		decoded := s.tryBase64Decode(string(content))

		// 调试日志
		decodedPreview := decoded
		if len(decodedPreview) > 200 {
			decodedPreview = decodedPreview[:200] + "..."
		}
		s.log("DEBUG", fmt.Sprintf("解码后内容长度: %d, 预览: %s", len(decoded), decodedPreview))

		nodeLinks := s.extractNodeLinks(decoded)

		// 记录类型统计
		s.logNodeTypeStats(url, nodeLinks)

		for _, link := range nodeLinks {
			allNodes = append(allNodes, map[string]interface{}{
				"url":        link,
				"source_url": url,
			})
		}
	}

	return allNodes, nil
}

// logNodeTypeStats 记录节点类型统计
func (s *ConfigUpdateService) logNodeTypeStats(url string, nodeLinks []string) {
	typeCount := make(map[string]int)
	for _, link := range nodeLinks {
		found := false
		for t := range supportedClashTypes {
			if strings.HasPrefix(link, t+"://") {
				typeCount[t]++
				found = true
				break
			}
		}
		if !found {
			// 简单检查其他协议
			if strings.HasPrefix(link, "hysteria2://") {
				typeCount["hysteria2"]++
			} else if strings.HasPrefix(link, "naive://") || strings.HasPrefix(link, "naive+https://") {
				typeCount["naive"]++
			} else if strings.HasPrefix(link, "anytls://") {
				typeCount["anytls"]++
			} else {
				typeCount["other"]++
			}
		}
	}

	var parts []string
	for t, c := range typeCount {
		parts = append(parts, fmt.Sprintf("%s:%d", t, c))
	}
	s.log("INFO", fmt.Sprintf("从 %s 提取到 %d 个节点链接 (%s)", url, len(nodeLinks), strings.Join(parts, ", ")))
}

// extractNodeLinks 提取节点链接
func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	var links []string
	var invalidLinks []string

	for _, re := range nodeLinkPatterns {
		matches := re.FindAllString(content, -1)
		for _, match := range matches {
			if s.isValidNodeLink(match) {
				links = append(links, match)
			} else {
				invalidLinks = append(invalidLinks, match)
			}
		}
	}

	if len(invalidLinks) > 0 {
		limit := 3
		if len(invalidLinks) < limit {
			limit = len(invalidLinks)
		}
		s.log("DEBUG", fmt.Sprintf("发现 %d 个无效链接，示例: %v", len(invalidLinks), invalidLinks[:limit]))
	}

	// 去重
	uniqueLinks := make(map[string]bool)
	var result []string
	for _, link := range links {
		if !uniqueLinks[link] {
			uniqueLinks[link] = true
			result = append(result, link)
		}
	}

	return result
}

// isValidNodeLink 验证节点链接是否完整有效
func (s *ConfigUpdateService) isValidNodeLink(link string) bool {
	link = strings.TrimSpace(link)
	if link == "" {
		return false
	}

	linkWithoutFragment := link
	if idx := strings.Index(link, "#"); idx != -1 {
		linkWithoutFragment = link[:idx]
	}

	if strings.HasPrefix(link, "ss://") {
		if strings.Contains(linkWithoutFragment, "@") {
			parts := strings.Split(linkWithoutFragment, "@")
			if len(parts) < 2 {
				return false
			}
			serverPart := parts[1]
			if idx := strings.Index(serverPart, "?"); idx != -1 {
				serverPart = serverPart[:idx]
			}
			if !strings.Contains(serverPart, ":") {
				return false
			}
		} else {
			encoded := strings.TrimPrefix(linkWithoutFragment, "ss://")
			if idx := strings.Index(encoded, "?"); idx != -1 {
				encoded = encoded[:idx]
			}
			if len(encoded) < 10 {
				return false
			}
		}
	} else if strings.HasPrefix(link, "vmess://") || strings.HasPrefix(link, "vless://") {
		encoded := strings.TrimPrefix(linkWithoutFragment, "vmess://")
		encoded = strings.TrimPrefix(encoded, "vless://")
		if idx := strings.Index(encoded, "?"); idx != -1 {
			encoded = encoded[:idx]
		}
		if len(encoded) < 10 {
			return false
		}
	} else if strings.HasPrefix(link, "trojan://") {
		if !strings.Contains(linkWithoutFragment, "@") {
			return false
		}
		parts := strings.Split(linkWithoutFragment, "@")
		if len(parts) < 2 {
			return false
		}
		serverPart := parts[1]
		if idx := strings.Index(serverPart, "?"); idx != -1 {
			serverPart = serverPart[:idx]
		}
		if !strings.Contains(serverPart, ":") {
			return false
		}
	} else if strings.HasPrefix(link, "ssr://") {
		encoded := strings.TrimPrefix(linkWithoutFragment, "ssr://")
		if len(encoded) < 10 {
			return false
		}
	} else if strings.HasPrefix(link, "hysteria://") || strings.HasPrefix(link, "hysteria2://") {
		if !strings.Contains(linkWithoutFragment, "@") && !strings.Contains(linkWithoutFragment, ":") {
			return false
		}
	} else if strings.HasPrefix(link, "tuic://") {
		if !strings.Contains(linkWithoutFragment, "@") {
			return false
		}
	}

	return true
}

// resolveRegion 从节点名称和服务器地址中解析地区信息
func (s *ConfigUpdateService) resolveRegion(name, server string) string {
	nameUpper := strings.ToUpper(name)
	// 按长度从长到短排序，优先匹配更长的关键词
	for _, kw := range regionKeys {
		if strings.Contains(nameUpper, strings.ToUpper(kw)) {
			if region, ok := regionMap[kw]; ok {
				return region
			}
		}
	}

	serverLower := strings.ToLower(server)
	for kw, region := range serverCodeMap {
		if strings.Contains(serverLower, kw) {
			return region
		}
	}

	return "未知"
}

// generateNodeDedupKey 生成节点去重键（统一格式：Type:Server:Port）
func (s *ConfigUpdateService) generateNodeDedupKey(nodeType, server string, port int) string {
	return fmt.Sprintf("%s:%s:%d", nodeType, server, port)
}

// ==========================================
// Database Operations
// ==========================================

// importNodesToDatabaseWithOrder 将节点导入到数据库的 nodes 表，并保存顺序索引
func (s *ConfigUpdateService) importNodesToDatabaseWithOrder(nodesWithOrder []nodeWithOrder) int {
	importedCount := 0

	for _, item := range nodesWithOrder {
		node := item.node
		orderIndex := item.orderIndex

		configJSON, _ := json.Marshal(node)
		configStr := string(configJSON)

		region := s.resolveRegion(node.Name, node.Server)

		var existingNode models.Node
		err := s.db.Where("type = ? AND name = ?", node.Type, node.Name).First(&existingNode).Error

		if err == nil {
			existingNode.Config = &configStr
			existingNode.Status = "online"
			existingNode.IsActive = true
			existingNode.OrderIndex = orderIndex
			existingNode.Region = region

			if err := s.db.Save(&existingNode).Error; err == nil {
				importedCount++
			} else {
				s.log("ERROR", fmt.Sprintf("更新节点失败: %s (%s), 错误: %v", node.Name, node.Type, err))
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			newNode := models.Node{
				Name:       node.Name,
				Type:       node.Type,
				Status:     "online",
				IsActive:   true,
				IsManual:   false,
				Config:     &configStr,
				Region:     region,
				OrderIndex: orderIndex,
			}
			if err := s.db.Create(&newNode).Error; err == nil {
				importedCount++
			} else {
				s.log("ERROR", fmt.Sprintf("创建节点失败: %s (%s), 错误: %v", node.Name, node.Type, err))
			}
		} else {
			s.log("ERROR", fmt.Sprintf("查询节点失败: %s (%s), 错误: %v", node.Name, node.Type, err))
		}
	}
	return importedCount
}

// fetchProxiesForUser 获取用户的可用节点
func (s *ConfigUpdateService) fetchProxiesForUser(user models.User, sub models.Subscription) ([]*ProxyNode, error) {
	var proxies []*ProxyNode
	processedNodes := make(map[string]bool)

	now := utils.GetBeijingTime()

	// 检查普通订阅是否过期
	isOrdExpired := !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now)

	// 计算专线到期时间
	// 如果设置了专线到期时间，以专线到期时间为准
	// 如果没设置专线到期时间，以普通线路到期时间为准
	var specialExpireTime time.Time
	hasSpecialExpireTime := false
	if user.SpecialNodeExpiresAt.Valid {
		specialExpireTime = utils.ToBeijingTime(user.SpecialNodeExpiresAt.Time)
		hasSpecialExpireTime = true
	} else if !sub.ExpireTime.IsZero() {
		specialExpireTime = utils.ToBeijingTime(sub.ExpireTime)
		hasSpecialExpireTime = true
	}
	isSpecialExpired := hasSpecialExpireTime && specialExpireTime.Before(now)

	// 根据用户的订阅类型决定是否包含普通节点
	// special_only: 只包含专线节点，不包含普通节点
	// both: 包含普通节点+专线节点，专线节点在最前面
	// 如果普通订阅过期，客户无法订阅普通线路（但可以订阅专线，如果专线未过期）
	includeNormalNodes := false
	if user.SpecialNodeSubscriptionType == "both" {
		// 全部订阅：只有普通订阅未过期时才包含普通节点
		includeNormalNodes = !isOrdExpired
	} else if user.SpecialNodeSubscriptionType == "special_only" {
		// 仅专线：不包含普通节点
		includeNormalNodes = false
	} else {
		// 默认情况：如果普通订阅未过期，包含普通节点
		includeNormalNodes = !isOrdExpired
	}

	if includeNormalNodes {
		// 获取普通节点
		var nodes []models.Node
		query := s.db.Model(&models.Node{}).Where("is_active = ?", true).Where("status != ?", "timeout")

		if err := query.Find(&nodes).Error; err != nil {
			return nil, err
		}

		for _, node := range nodes {
			proxyNodes, err := s.parseNodeToProxies(&node)
			if err != nil {
				continue
			}

			for _, proxy := range proxyNodes {
				// 使用统一的去重键生成函数
				key := s.generateNodeDedupKey(proxy.Type, proxy.Server, proxy.Port)
				if processedNodes[key] {
					continue
				}
				processedNodes[key] = true
				proxies = append(proxies, proxy)
			}
		}
	}

	// 获取专属节点（专线节点始终在最前面）
	var customNodes []models.CustomNode
	if err := s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", user.ID, true).
		Find(&customNodes).Error; err == nil {

		var customProxies []*ProxyNode
		for _, cn := range customNodes {
			// 判断专线节点是否过期
			// 1. 如果节点设置了 FollowUserExpire，使用用户的专线到期时间（或普通到期时间）
			// 2. 如果节点设置了 ExpireTime，使用节点的到期时间
			// 3. 如果都没设置，使用用户的专线到期时间（或普通到期时间）
			isSpecNodeExpired := false
			if cn.FollowUserExpire {
				// 跟随用户到期时间
				isSpecNodeExpired = isSpecialExpired
			} else if cn.ExpireTime != nil {
				// 使用节点自己的到期时间
				expireTimeBeijing := utils.ToBeijingTime(*cn.ExpireTime)
				isSpecNodeExpired = expireTimeBeijing.Before(now)
			} else {
				// 默认使用用户的专线到期时间（或普通到期时间）
				isSpecNodeExpired = isSpecialExpired
			}

			if isSpecNodeExpired || cn.Status == "timeout" {
				continue
			}

			displayName := cn.DisplayName
			if displayName == "" {
				displayName = "专线-" + cn.Name
			}

			if cn.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(cn.Config), &proxyNode); err == nil {
					proxyNode.Name = displayName
					customProxies = append(customProxies, &proxyNode)
				}
			}
		}

		// 将专线节点放在最前面
		proxies = append(customProxies, proxies...)
	}

	return proxies, nil
}

// parseNodeToProxies 解析数据库节点模型为代理节点对象
func (s *ConfigUpdateService) parseNodeToProxies(node *models.Node) ([]*ProxyNode, error) {
	if node.Config != nil && *node.Config != "" {
		var configProxy ProxyNode
		if err := json.Unmarshal([]byte(*node.Config), &configProxy); err == nil {
			configProxy.Name = node.Name
			return []*ProxyNode{&configProxy}, nil
		}
	}
	return nil, fmt.Errorf("节点配置为空")
}

// getSubscriptionContext 获取订阅上下文
func (s *ConfigUpdateService) getSubscriptionContext(token string, clientIP string, userAgent string) *SubscriptionContext {
	ctx := &SubscriptionContext{
		Status: StatusNotFound,
	}

	// 1. 查找订阅
	var sub models.Subscription
	if err := s.db.Where("subscription_url = ?", token).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var reset models.SubscriptionReset
			if err := s.db.Where("old_subscription_url = ?", token).First(&reset).Error; err == nil {
				ctx.Status = StatusOldAddress
				ctx.ResetRecord = &reset
				return ctx
			}
		}
		return ctx
	}
	ctx.Subscription = sub

	// 2. 查找用户
	var user models.User
	if err := s.db.First(&user, sub.UserID).Error; err != nil {
		return ctx
	}
	ctx.User = user

	// 3. 检查状态
	if !user.IsActive {
		ctx.Status = StatusAccountAbnormal
		return ctx
	}
	if !sub.IsActive || sub.Status != "active" {
		ctx.Status = StatusInactive
		return ctx
	}
	// 检查订阅是否过期
	// SQLite 存储的时间格式是 UTC (如: 2027-01-22 00:00:00+00:00)
	// 我们需要统一使用 UTC 时间进行比较，避免时区问题
	if !sub.ExpireTime.IsZero() {
		// 将 ExpireTime 转换为 UTC（如果它还不是 UTC）
		expireTimeUTC := sub.ExpireTime.UTC()
		// 获取当前 UTC 时间
		nowUTC := time.Now().UTC()

		// 调试日志：记录时间比较信息
		if utils.AppLogger != nil {
			utils.AppLogger.Info("订阅过期检查 - SubscriptionID=%d, UserID=%d, ExpireTime(原始)=%s, ExpireTime(UTC)=%s, Now(UTC)=%s, ExpireTime.Unix=%d, Now.Unix=%d, Before=%v",
				sub.ID, sub.UserID,
				sub.ExpireTime.Format("2006-01-02 15:04:05 MST"),
				expireTimeUTC.Format("2006-01-02 15:04:05 MST"),
				nowUTC.Format("2006-01-02 15:04:05 MST"),
				expireTimeUTC.Unix(),
				nowUTC.Unix(),
				expireTimeUTC.Before(nowUTC))
		}

		// 使用 UTC 时间进行比较
		if expireTimeUTC.Before(nowUTC) {
			ctx.Status = StatusExpired
			return ctx
		}
	}

	// 4. 检查设备
	var currentDevices int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&currentDevices)
	ctx.CurrentDevices = int(currentDevices)
	ctx.DeviceLimit = sub.DeviceLimit

	// 设备限制检查：如果限制为0，不允许使用
	if sub.DeviceLimit == 0 {
		ctx.Status = StatusDeviceOverLimit
		return ctx
	}

	// 如果设备数量达到或超过限制，检查当前设备是否已存在
	if sub.DeviceLimit > 0 && int(currentDevices) >= sub.DeviceLimit {
		var device models.Device
		isKnownDevice := false
		if err := s.db.Where("subscription_id = ? AND ip_address = ? AND user_agent = ?", sub.ID, clientIP, userAgent).First(&device).Error; err == nil {
			isKnownDevice = true
		}
		if !isKnownDevice {
			ctx.Status = StatusDeviceOverLimit
			return ctx
		}
	}

	// 5. 获取节点
	proxies, err := s.fetchProxiesForUser(user, sub)
	if err != nil {
		ctx.Proxies = []*ProxyNode{}
	} else {
		ctx.Proxies = proxies
	}

	ctx.Status = StatusNormal
	return ctx
}

// UpdateSubscriptionConfig 更新订阅配置
func (s *ConfigUpdateService) UpdateSubscriptionConfig(subscriptionURL string) error {
	var count int64
	s.db.Model(&models.Subscription{}).Where("subscription_url = ?", subscriptionURL).Count(&count)
	if count == 0 {
		return fmt.Errorf("订阅不存在")
	}
	return nil
}

// ==========================================
// Config Generation
// ==========================================

// GenerateClashConfig 生成 Clash 配置
func (s *ConfigUpdateService) GenerateClashConfig(token string, clientIP string, userAgent string) (string, error) {
	// 每次生成配置前都刷新系统配置，确保使用最新的域名设置
	s.refreshSystemConfig()

	ctx := s.getSubscriptionContext(token, clientIP, userAgent)

	if ctx.Status != StatusNormal {
		errorNodes := s.generateErrorNodes(ctx.Status, ctx)
		return s.generateClashYAML(errorNodes), nil
	}

	finalNodes := s.addInfoNodes(ctx.Proxies, ctx)
	return s.generateClashYAML(finalNodes), nil
}

// GenerateUniversalConfig 生成通用订阅配置
func (s *ConfigUpdateService) GenerateUniversalConfig(token string, clientIP string, userAgent string, format string) (string, error) {
	// 每次生成配置前都刷新系统配置，确保使用最新的域名设置
	s.refreshSystemConfig()

	ctx := s.getSubscriptionContext(token, clientIP, userAgent)
	var nodesToExport []*ProxyNode

	if ctx.Status != StatusNormal {
		nodesToExport = s.generateErrorNodes(ctx.Status, ctx)
	} else {
		nodesToExport = s.addInfoNodes(ctx.Proxies, ctx)
	}

	var links []string
	for _, node := range nodesToExport {
		var link string
		if format == "ssr" && node.Type == "ssr" {
			link = s.nodeToSSRLink(node)
		} else {
			link = s.nodeToLink(node)
		}
		if link != "" {
			links = append(links, link)
		}
	}

	return base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n"))), nil
}

// generateClashYAML 生成 Clash YAML 配置
func (s *ConfigUpdateService) generateClashYAML(proxies []*ProxyNode) string {
	var builder strings.Builder

	// 过滤支持的节点
	filteredProxies := make([]*ProxyNode, 0)
	for _, proxy := range proxies {
		if supportedClashTypes[proxy.Type] {
			filteredProxies = append(filteredProxies, proxy)
		}
	}

	// 基础配置
	builder.WriteString("port: 7890\n")
	builder.WriteString("socks-port: 7891\n")
	builder.WriteString("allow-lan: true\n")
	builder.WriteString("mode: Rule\n")
	builder.WriteString("log-level: info\n")
	builder.WriteString("external-controller: 127.0.0.1:9090\n\n")

	builder.WriteString("proxies:\n")

	// 确保节点名称唯一
	usedNames := make(map[string]bool)
	var proxyNames []string

	for _, proxy := range filteredProxies {
		originalName := proxy.Name
		newName := originalName
		counter := 1
		for usedNames[newName] {
			newName = fmt.Sprintf("%s_%d", originalName, counter)
			counter++
		}
		proxy.Name = newName
		usedNames[newName] = true

		builder.WriteString(s.nodeToYAML(proxy, 2))
		proxyNames = append(proxyNames, s.escapeYAMLString(proxy.Name))
	}

	// 代理组
	builder.WriteString("\nproxy-groups:\n")

	// 节点选择
	builder.WriteString("  - name: \"🚀 节点选择\"\n")
	builder.WriteString("    type: select\n")
	builder.WriteString("    proxies:\n")
	builder.WriteString("      - \"♻️ 自动选择\"\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	// 自动选择
	builder.WriteString("  - name: \"♻️ 自动选择\"\n")
	builder.WriteString("    type: url-test\n")
	builder.WriteString("    url: http://www.gstatic.com/generate_204\n")
	builder.WriteString("    interval: 300\n")
	builder.WriteString("    tolerance: 50\n")
	builder.WriteString("    proxies:\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	// 规则
	builder.WriteString("\nrules:\n")
	builder.WriteString("  - DOMAIN-SUFFIX,local,DIRECT\n")
	builder.WriteString("  - IP-CIDR,127.0.0.0/8,DIRECT\n")
	builder.WriteString("  - IP-CIDR,172.16.0.0/12,DIRECT\n")
	builder.WriteString("  - IP-CIDR,192.168.0.0/16,DIRECT\n")
	builder.WriteString("  - GEOIP,CN,DIRECT\n")
	builder.WriteString("  - MATCH,🚀 节点选择\n")

	return builder.String()
}

// addInfoNodes 添加信息节点
func (s *ConfigUpdateService) addInfoNodes(proxies []*ProxyNode, ctx *SubscriptionContext) []*ProxyNode {
	// 确保配置已刷新（已在 GenerateClashConfig 和 GenerateUniversalConfig 中刷新）

	expireTimeStr := "无限期"
	if !ctx.Subscription.ExpireTime.IsZero() {
		expireTimeStr = ctx.Subscription.ExpireTime.Format("2006-01-02")
	}

	infoNodes := []*ProxyNode{
		{
			Name:     fmt.Sprintf("📢 官网: %s", s.siteURL),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		},
		{
			Name:     fmt.Sprintf("⏰ 到期: %s", expireTimeStr),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		},
		{
			Name:     fmt.Sprintf("📱 设备: %d/%d", ctx.CurrentDevices, ctx.DeviceLimit),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		},
	}

	// 如果配置了客服QQ，添加客服QQ信息节点
	if s.supportQQ != "" {
		infoNodes = append(infoNodes, &ProxyNode{
			Name:     fmt.Sprintf("💬 客服QQ: %s", s.supportQQ),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		})
	}

	return append(infoNodes, proxies...)
}

// generateErrorNodes 生成错误提示节点
func (s *ConfigUpdateService) generateErrorNodes(status SubscriptionStatus, ctx *SubscriptionContext) []*ProxyNode {
	var reason, solution string

	switch status {
	case StatusExpired:
		reason = "订阅已过期"
		solution = fmt.Sprintf("请前往官网续费 (过期时间: %s)", ctx.Subscription.ExpireTime.Format("2006-01-02"))
	case StatusInactive:
		reason = "订阅已失效"
		solution = "请联系管理员检查订阅状态"
	case StatusAccountAbnormal:
		reason = "账户异常"
		solution = "您的账户状态异常或已被禁用，请联系客服"
	case StatusDeviceOverLimit:
		reason = "设备数量超限"
		solution = fmt.Sprintf("当前设备 %d/%d，请在官网删除不使用的设备", ctx.CurrentDevices, ctx.DeviceLimit)
	case StatusOldAddress:
		reason = "订阅地址已变更"
		solution = "请登录官网获取最新的订阅地址"
	case StatusNotFound:
		reason = "订阅不存在"
		solution = "请检查订阅链接是否正确，或重新复制"
	default:
		reason = "账户异常"
		solution = "检测到账户异常，请联系管理员"
	}

	// 确保配置已刷新（已在 GenerateClashConfig 和 GenerateUniversalConfig 中刷新）
	return []*ProxyNode{
		{
			Name:     fmt.Sprintf("📢 官网: %s", s.siteURL),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
		{
			Name:     fmt.Sprintf("❌ 原因: %s", reason),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
		{
			Name:     fmt.Sprintf("💡 解决: %s", solution),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
		{
			Name: func() string {
				if s.supportQQ != "" {
					return fmt.Sprintf("💬 客服QQ: %s", s.supportQQ)
				}
				return "💬 客服QQ: 请在系统设置中配置"
			}(),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
	}
}

// nodeToYAML 将节点转换为 YAML 格式
func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	indentStr := strings.Repeat(" ", indent)
	var builder strings.Builder

	escapedName := s.escapeYAMLString(node.Name)

	builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, escapedName))
	builder.WriteString(fmt.Sprintf("%s  type: %s\n", indentStr, node.Type))
	builder.WriteString(fmt.Sprintf("%s  server: %s\n", indentStr, node.Server))
	builder.WriteString(fmt.Sprintf("%s  port: %d\n", indentStr, node.Port))

	switch node.Type {
	case "ss":
		if node.Cipher != "" {
			builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
		}
		if node.Password != "" {
			builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
		}
	case "vmess":
		if node.UUID != "" {
			builder.WriteString(fmt.Sprintf("%s  uuid: %s\n", indentStr, node.UUID))
		}
		if alterId, ok := node.Options["alterId"]; !ok {
			builder.WriteString(fmt.Sprintf("%s  alterId: 0\n", indentStr))
		} else {
			builder.WriteString(fmt.Sprintf("%s  alterId: %v\n", indentStr, alterId))
		}
		if node.Cipher == "" {
			node.Cipher = "auto"
		}
		builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
	case "vless":
		if node.UUID != "" {
			builder.WriteString(fmt.Sprintf("%s  uuid: %s\n", indentStr, node.UUID))
		}
	case "trojan":
		if node.Password != "" {
			builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
		}
	case "ssr":
		if node.Cipher != "" {
			builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
		}
		if node.Password != "" {
			builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
		}
	}

	if node.TLS {
		builder.WriteString(fmt.Sprintf("%s  tls: true\n", indentStr))
	}
	if node.Network != "" && node.Network != "tcp" {
		builder.WriteString(fmt.Sprintf("%s  network: %s\n", indentStr, node.Network))
	}
	if node.UDP {
		builder.WriteString(fmt.Sprintf("%s  udp: true\n", indentStr))
	}

	// 写入 Options
	optionsIndentStr := indentStr + "  "

	// 对 Options key 进行排序以保证输出稳定
	var keys []string
	for k := range node.Options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Options[key]
		if key == "alterId" && node.Type == "vmess" {
			continue
		}
		s.writeYAMLValue(&builder, optionsIndentStr, key, value, 2)
	}

	return builder.String()
}

// writeYAMLValue 递归写入 YAML 值
func (s *ConfigUpdateService) writeYAMLValue(builder *strings.Builder, indentStr, key string, value interface{}, indentLevel int) {
	escapedKey := s.escapeYAMLString(key)

	switch v := value.(type) {
	case map[string]interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndentStr := indentStr + "  "

		// 特殊处理 http-opts
		if key == "http-opts" {
			s.writeHTTPOpts(builder, subIndentStr, v)
			return
		}

		for k, val := range v {
			if strMap, ok := val.(map[string]string); ok {
				escapedK := s.escapeYAMLString(k)
				builder.WriteString(fmt.Sprintf("%s%s:\n", subIndentStr, escapedK))
				subSubIndentStr := subIndentStr + "  "
				for k2, v2 := range strMap {
					escapedK2 := s.escapeYAMLString(k2)
					escapedV2 := s.escapeYAMLString(v2)
					builder.WriteString(fmt.Sprintf("%s%s: %s\n", subSubIndentStr, escapedK2, escapedV2))
				}
			} else {
				s.writeYAMLValue(builder, subIndentStr, k, val, indentLevel+1)
			}
		}
	case []interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndentStr := indentStr + "  "
		for _, item := range v {
			builder.WriteString(fmt.Sprintf("%s- ", subIndentStr))
			s.writeYAMLValueInline(builder, item)
			builder.WriteString("\n")
		}
	case []string:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndentStr := indentStr + "  "
		for _, item := range v {
			escapedItem := s.escapeYAMLString(item)
			builder.WriteString(fmt.Sprintf("%s- %s\n", subIndentStr, escapedItem))
		}
	default:
		escapedVal := s.escapeYAMLString(fmt.Sprintf("%v", v))
		builder.WriteString(fmt.Sprintf("%s%s: %s\n", indentStr, escapedKey, escapedVal))
	}
}

// writeHTTPOpts 辅助写入 http-opts
func (s *ConfigUpdateService) writeHTTPOpts(builder *strings.Builder, indentStr string, v map[string]interface{}) {
	for k, val := range v {
		if k == "path" {
			s.writeYAMLList(builder, indentStr, k, val)
		} else if k == "headers" {
			escapedK := s.escapeYAMLString(k)
			builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedK))
			subIndentStr := indentStr + "  "
			if headersMap, ok := val.(map[string]interface{}); ok {
				for hk, hv := range headersMap {
					s.writeYAMLList(builder, subIndentStr, hk, hv)
				}
			}
		}
	}
}

// writeYAMLList 辅助写入列表配置
func (s *ConfigUpdateService) writeYAMLList(builder *strings.Builder, indentStr, key string, val interface{}) {
	escapedK := s.escapeYAMLString(key)
	builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedK))
	subIndentStr := indentStr + "  "

	writeItem := func(item interface{}) {
		escapedItem := s.escapeYAMLString(fmt.Sprintf("%v", item))
		builder.WriteString(fmt.Sprintf("%s- %s\n", subIndentStr, escapedItem))
	}

	if str, ok := val.(string); ok {
		writeItem(str)
	} else if slice, ok := val.([]string); ok {
		for _, item := range slice {
			writeItem(item)
		}
	} else if slice, ok := val.([]interface{}); ok {
		for _, item := range slice {
			writeItem(item)
		}
	}
}

// writeYAMLValueInline 内联写入 YAML 值
func (s *ConfigUpdateService) writeYAMLValueInline(builder *strings.Builder, value interface{}) {
	switch v := value.(type) {
	case string:
		builder.WriteString(s.escapeYAMLString(v))
	case int, int64, float64, bool:
		builder.WriteString(fmt.Sprintf("%v", v))
	default:
		builder.WriteString(s.escapeYAMLString(fmt.Sprintf("%v", v)))
	}
}

// escapeYAMLString 转义 YAML 字符串
func (s *ConfigUpdateService) escapeYAMLString(str string) string {
	if str == "" {
		return "\"\""
	}
	needsQuotes := false
	specialChars := []string{":", "\"", "'", "\n", "\r", "\t", "#", "@", "&", "*", "?", "|", ">", "!", "%", "`", "[", "]", "{", "}", ","}
	for _, char := range specialChars {
		if strings.Contains(str, char) {
			needsQuotes = true
			break
		}
	}
	if strings.HasPrefix(str, " ") || strings.HasSuffix(str, " ") {
		needsQuotes = true
	}
	if needsQuotes {
		escaped := strings.ReplaceAll(str, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return str
}

// ==========================================
// Utils & Helpers
// ==========================================

// nodeToLink 将节点转换为通用链接
func (s *ConfigUpdateService) nodeToLink(node *ProxyNode) string {
	switch node.Type {
	case "vmess":
		return s.vmessToLink(node)
	case "vless":
		return s.vlessToLink(node)
	case "trojan":
		return s.trojanToLink(node)
	case "ss":
		return s.shadowsocksToLink(node)
	case "ssr":
		return s.nodeToSSRLink(node)
	default:
		return ""
	}
}

func (s *ConfigUpdateService) vmessToLink(proxy *ProxyNode) string {
	data := map[string]interface{}{
		"v":    "2",
		"ps":   proxy.Name,
		"add":  proxy.Server,
		"port": proxy.Port,
		"id":   proxy.UUID,
		"net":  proxy.Network,
		"type": "none",
	}

	if proxy.TLS {
		data["tls"] = "tls"
	}

	if proxy.Options != nil {
		if wsOpts, ok := proxy.Options["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok {
				data["path"] = path
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					data["host"] = host
				}
			}
		}
	}

	jsonData, _ := json.Marshal(data)
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded
}

func (s *ConfigUpdateService) vlessToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "vless",
		User:     url.User(proxy.UUID),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Network != "" {
		q.Set("type", proxy.Network)
	}
	if proxy.TLS {
		q.Set("security", "tls")
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ConfigUpdateService) trojanToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "trojan",
		User:     url.User(proxy.Password),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}
	return u.String()
}

func (s *ConfigUpdateService) shadowsocksToLink(proxy *ProxyNode) string {
	auth := fmt.Sprintf("%s:%s", proxy.Cipher, proxy.Password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	u := &url.URL{
		Scheme:   "ss",
		User:     url.User(encoded),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}
	return u.String()
}

func (s *ConfigUpdateService) nodeToSSRLink(node *ProxyNode) string {
	if node.Type != "ssr" && node.Type != "ss" {
		return ""
	}

	getString := func(opts map[string]interface{}, key, defaultValue string) string {
		if v, ok := opts[key].(string); ok {
			return v
		}
		return defaultValue
	}

	server := node.Server
	port := node.Port
	protocol := getString(node.Options, "protocol", "origin")
	method := node.Cipher
	obfs := getString(node.Options, "obfs", "plain")
	password := base64.RawURLEncoding.EncodeToString([]byte(node.Password))

	obfsparam := base64.RawURLEncoding.EncodeToString([]byte(getString(node.Options, "obfs-param", "")))
	protoparam := base64.RawURLEncoding.EncodeToString([]byte(getString(node.Options, "protocol-param", "")))
	remarks := base64.RawURLEncoding.EncodeToString([]byte(node.Name))
	group := base64.RawURLEncoding.EncodeToString([]byte("GoWeb"))

	ssrStr := fmt.Sprintf("%s:%d:%s:%s:%s:%s/?obfsparam=%s&protoparam=%s&remarks=%s&group=%s",
		server, port, protocol, method, obfs, password,
		obfsparam, protoparam, remarks, group)

	return "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(ssrStr))
}

func (s *ConfigUpdateService) tryBase64Decode(text string) string {
	cleanText := strings.ReplaceAll(text, " ", "")
	cleanText = strings.ReplaceAll(cleanText, "\n", "")
	cleanText = strings.ReplaceAll(cleanText, "\r", "")
	cleanText = strings.ReplaceAll(cleanText, "-", "+")
	cleanText = strings.ReplaceAll(cleanText, "_", "/")

	if len(cleanText)%4 != 0 {
		cleanText += strings.Repeat("=", 4-len(cleanText)%4)
	}

	decoded, err := base64.StdEncoding.DecodeString(cleanText)
	if err != nil {
		return text
	}
	return string(decoded)
}
