// Package activity 判断「用户到底有没有在用」，供后台用户/设备列表显示使用状态。
//
// 背景与边界（务必先读）：
//   - 自研客户端可以发真心跳（前台/后台、进程存活），那是**高置信度**信号；
//   - 第三方客户端（v2rayN / Clash / Shadowrocket / sing-box…）**不会**、也无法给面板发心跳，
//     因此只能被动推断：① 它来拉订阅了（说明配置还在更新）② 节点侧按用户上报了连接与流量
//     （说明此刻真在连）。两者都不是「客户端主动心跳」，产品上要如实展示判定依据，
//     否则很容易被当成「在线统计不准」。
//
// 状态定义（StateFor 返回）：
//
//	online  最近 OnlineTTL 内节点侧上报连接数 > 0 —— 真·在线（需节点侧能区分用户）
//	active  最近 ActiveWindow 内有任何信号（拉订阅 / 节点上报）—— 在用
//	idle    最近 IdleWindow 内有信号 —— 近期用过，现在没动静
//	silent  超过 IdleWindow 无任何信号（或从未）—— 静默
//
// 实时数据放 Redis（zset 存时间戳，天然支持「最近 N 分钟有信号」的集合运算），
// Redis 不可用时自动降级为进程内存储（功能可用，但多实例不共享、重启即失效）。
package activity

import (
	"encoding/json"
	"log"
	"time"

	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"

	"gorm.io/gorm"
)

// 状态取值
const (
	StateOnline = "online"
	StateActive = "active"
	StateIdle   = "idle"
	StateSilent = "silent"
)

// 时间窗口。做成变量（而不是常量）便于测试注入与后续改成后台可配。
var (
	// OnlineTTL 节点侧上报连接数后，多久之内仍算「在线」
	OnlineTTL = 90 * time.Second
	// ActiveWindow 多久之内有信号算「活跃」
	ActiveWindow = 15 * time.Minute
	// IdleWindow 多久之内有信号算「近期用过」（超过则静默）
	IdleWindow = 24 * time.Hour
	// persistThrottle 同一用户最快多久往数据库落一次快照
	persistThrottle = 60 * time.Second
	// metaTTL 订阅元信息（UA/IP/客户端）保留时长
	metaTTL = 7 * 24 * time.Hour
)

// 存储键（统一前缀，便于运维清理；菜单 4 的按前缀删除也能覆盖）
const (
	keyActive = "cboard:act:active" // zset: member=userID, score=unix（任何信号）
	keyOnline = "cboard:act:online" // zset: member=userID, score=unix（节点上报 conns>0）
	keyConns  = "cboard:act:conns:" // string: 连接数，TTL=OnlineTTL
	keyMeta   = "cboard:act:meta:"  // string: Meta 的 JSON，TTL=metaTTL
	keyLock   = "cboard:act:lock:"  // string: 落库节流锁
)

// DefaultProfileUpdateIntervalMinutes 订阅响应里下发的客户端更新间隔（分钟）。
// 这是我们在客户端侧唯一能「加密采样」的旋钮：第三方客户端会按这个间隔回拉订阅。
const DefaultProfileUpdateIntervalMinutes = 60

// Meta 订阅侧元信息
type Meta struct {
	UA            string    `json:"ua,omitempty"`
	IP            string    `json:"ip,omitempty"`
	Location      string    `json:"location,omitempty"`
	Format        string    `json:"format,omitempty"`
	Software      string    `json:"software,omitempty"`
	SubFetchCount int64     `json:"sub_fetch_count"`
	LastSubAt     time.Time `json:"last_sub_at,omitempty"`
}

// State 一个用户的使用状态
type State struct {
	State            string    `json:"state"`
	OnlineConns      int       `json:"online_conns"`
	LastActiveAt     time.Time `json:"last_active_at,omitempty"`
	LastSubAt        time.Time `json:"last_sub_at,omitempty"`
	LastNodeReportAt time.Time `json:"last_node_report_at,omitempty"`
	SubFetchCount    int64     `json:"sub_fetch_count"`
	UA               string    `json:"ua,omitempty"`
	IP               string    `json:"ip,omitempty"`
	Location         string    `json:"location,omitempty"`
	Format           string    `json:"format,omitempty"`
	Software         string    `json:"software,omitempty"`
}

// UsageCounts 全局计数（后台概览用）
type UsageCounts struct {
	Online int64 `json:"online"`
	Active int64 `json:"active"`
	Idle   int64 `json:"idle"`
}

// NodeUserStat 节点侧上报的单个用户状态
type NodeUserStat struct {
	UserID uint
	Conns  int
	Up     int64
	Down   int64
}

// ---------------- 存储抽象：Redis 优先，内存兜底 ----------------

type store interface {
	zadd(key string, member uint, score int64)
	zrem(key string, member uint)
	membersSince(key string, since int64) []uint
	zcountSince(key string, since int64) int64
	scoreOf(key string, member uint) (int64, bool)
	setStr(key, val string, ttl time.Duration)
	getStr(key string) (string, bool)
	setNX(key string, ttl time.Duration) bool
	del(keys ...string)
}

// GetStore 返回当前生效的存储实现。Redis 可能在后端启动之后才连通
// （例如菜单 12 配置完 Redis 再重启），所以每次调用都重新判断，可用即自动切换。
func GetStore() store {
	storeMu.Lock()
	defer storeMu.Unlock()
	if storeImpl == nil {
		if cache.IsRedisEnabled() {
			storeImpl = &redisStore{}
		} else {
			storeImpl = newMemStore()
		}
		return storeImpl
	}
	if cache.IsRedisEnabled() {
		if _, ok := storeImpl.(*redisStore); !ok {
			storeImpl = &redisStore{}
		}
	} else if _, ok := storeImpl.(*memStore); !ok {
		storeImpl = newMemStore()
	}
	return storeImpl
}

// ResetStoreForTest 仅测试使用：丢弃当前实现，强制下次重新选择
func ResetStoreForTest() {
	storeMu.Lock()
	defer storeMu.Unlock()
	storeImpl = nil
}

// ---------------- 对外 API ----------------

// MarkSubscriptionFetch 记录一次订阅拉取（第三方客户端最可靠的「还活着」信号）。
// 非阻塞：Redis 写是微秒级；落库走节流 + 后台协程，绝不拖慢订阅响应。
func MarkSubscriptionFetch(userID uint, ua, ip, format, software string) {
	if userID == 0 {
		return
	}
	s := GetStore()
	now := time.Now()
	s.zadd(keyActive, userID, now.Unix())

	prev := Meta{}
	if raw, ok := s.getStr(keyMeta + itoa(userID)); ok {
		_ = json.Unmarshal([]byte(raw), &prev)
	}
	location := prev.Location
	if ip != "" && ip != prev.IP {
		if loc := geoip.GetLocationWithCache(ip); loc.Valid && loc.String != "" {
			location = loc.String
		}
	}
	m := Meta{
		UA:            truncate(ua, 255),
		IP:            ip,
		Location:      location,
		Format:        format,
		Software:      software,
		SubFetchCount: prev.SubFetchCount + 1,
		LastSubAt:     now,
	}
	if b, err := json.Marshal(m); err == nil {
		s.setStr(keyMeta+itoa(userID), string(b), metaTTL)
	}
	persistThrottled(userID, stateFromMeta(m))
}

// MarkNodeUsers 记录节点侧按用户上报的连接数与流量（真·在线信号）。
// 节点当前是单凭据设计时该数组为空，本函数自然不会被触发；节点升级为多用户后无需再改面板。
func MarkNodeUsers(stats []NodeUserStat) {
	if len(stats) == 0 {
		return
	}
	now := time.Now()
	nowUnix := now.Unix()
	s := GetStore()
	for _, st := range stats {
		if st.UserID == 0 {
			continue
		}
		// 任何上报都算活动信号（conns=0 也说明节点还认这个用户）
		s.zadd(keyActive, st.UserID, nowUnix)
		if st.Conns > 0 {
			s.zadd(keyOnline, st.UserID, nowUnix)
			s.setStr(keyConns+itoa(st.UserID), itoa64(int64(st.Conns)), OnlineTTL)
		} else {
			// 节点明确上报 conns=0：必须把 online 集合里的旧记录也删掉，
			// 否则 90 秒窗口内仍会被判成"在线"（单元测试抓到的真实缺陷）
			s.zrem(keyOnline, st.UserID)
			s.del(keyConns + itoa(st.UserID))
		}
		go persistNodeReport(st, now)
	}
}

// StateFor 单个用户的状态
func StateFor(userID uint) State {
	return StatesFor([]uint{userID})[userID]
}

// StatesFor 批量取状态（后台列表用，避免 N 次往返）
func StatesFor(userIDs []uint) map[uint]State {
	out := make(map[uint]State, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	now := time.Now()
	s := GetStore()
	onlineSince := now.Add(-OnlineTTL).Unix()

	online := make(map[uint]struct{})
	for _, id := range s.membersSince(keyOnline, onlineSince) {
		online[id] = struct{}{}
	}

	for _, id := range userIDs {
		st := State{State: StateSilent}
		if ts, ok := s.scoreOf(keyActive, id); ok {
			st.LastActiveAt = time.Unix(ts, 0)
			age := now.Sub(st.LastActiveAt)
			switch {
			case age <= ActiveWindow:
				st.State = StateActive
			case age <= IdleWindow:
				st.State = StateIdle
			}
		}
		if raw, ok := s.getStr(keyMeta + itoa(id)); ok {
			var m Meta
			if json.Unmarshal([]byte(raw), &m) == nil {
				st.UA, st.IP, st.Location, st.Format, st.Software = m.UA, m.IP, m.Location, m.Format, m.Software
				st.SubFetchCount = m.SubFetchCount
				st.LastSubAt = m.LastSubAt
			}
		}
		if _, ok := online[id]; ok {
			st.State = StateOnline
			if ts, ok2 := s.scoreOf(keyOnline, id); ok2 {
				st.LastNodeReportAt = time.Unix(ts, 0)
			}
			if c, ok2 := s.getStr(keyConns + itoa(id)); ok2 {
				st.OnlineConns = atoiDef(c, 0)
			}
		}
		// Redis 里没有（例如重启/换机/Redis 被清）→ 用数据库快照兜底
		if st.State == StateSilent && st.LastSubAt.IsZero() {
			if row, err := loadPersisted(id); err == nil && row != nil {
				applyPersisted(&st, row, now)
			}
		}
		out[id] = st
	}
	return out
}

// Counts 在线 / 活跃 / 近期用过 的用户数
func Counts() UsageCounts {
	now := time.Now()
	s := GetStore()
	return UsageCounts{
		Online: s.zcountSince(keyOnline, now.Add(-OnlineTTL).Unix()),
		Active: s.zcountSince(keyActive, now.Add(-ActiveWindow).Unix()),
		Idle:   s.zcountSince(keyActive, now.Add(-IdleWindow).Unix()),
	}
}

// FilterIDs 按状态筛选用户 ID（供后台列表做 SQL 过滤，避免打乱分页）。
// 返回 (ids, true)；状态为 silent 时请配合 SilentIDs() 使用 NOT IN。
func FilterIDs(state string) []uint {
	now := time.Now()
	s := GetStore()
	switch state {
	case StateOnline:
		return s.membersSince(keyOnline, now.Add(-OnlineTTL).Unix())
	case StateActive:
		return s.membersSince(keyActive, now.Add(-ActiveWindow).Unix())
	case StateIdle:
		recent := s.membersSince(keyActive, now.Add(-IdleWindow).Unix())
		hot := make(map[uint]struct{})
		for _, id := range s.membersSince(keyActive, now.Add(-ActiveWindow).Unix()) {
			hot[id] = struct{}{}
		}
		out := make([]uint, 0, len(recent))
		for _, id := range recent {
			if _, ok := hot[id]; !ok {
				out = append(out, id)
			}
		}
		return out
	default:
		return nil
	}
}

// SilentIDs 返回 IdleWindow 内有信号的用户 ID（配合 SQL NOT IN 得到「静默」用户）
func SilentIDs() []uint {
	now := time.Now()
	return GetStore().membersSince(keyActive, now.Add(-IdleWindow).Unix())
}

// ---------------- 内部实现 ----------------

func stateFromMeta(m Meta) State {
	return State{
		State:         StateActive,
		LastActiveAt:  m.LastSubAt,
		LastSubAt:     m.LastSubAt,
		SubFetchCount: m.SubFetchCount,
		UA:            m.UA,
		IP:            m.IP,
		Location:      m.Location,
		Format:        m.Format,
		Software:      m.Software,
	}
}

// persistThrottled 落库节流：同一用户 persistThrottle 内只写一次数据库。
// 取舍说明：数据库只是「重启后的兜底快照」，实时状态由 Redis 提供，
// 因此允许快照最多滞后 persistThrottle；换来的是拉订阅这条高频路径不写库。
func persistThrottled(userID uint, st State) {
	if !GetStore().setNX(keyLock+itoa(userID), persistThrottle) {
		return
	}
	go persist(userID, st)
}

func persist(userID uint, st State) {
	db := database.GetDB()
	if db == nil || userID == 0 {
		return
	}
	// 仓库既有教训：clause.OnConflict 必须与唯一约束完全匹配才生效，
	// 这里沿用项目惯用的 FirstOrInit + Save，避免又踩一次。
	var row models.UserUsageState
	if err := db.Where("user_id = ?", userID).FirstOrInit(&row).Error; err != nil {
		log.Printf("activity: 读取活动记录失败 user=%d: %v", userID, err)
		return
	}
	row.UserID = userID
	row.SubFetchCount = st.SubFetchCount
	if !st.LastSubAt.IsZero() {
		t := st.LastSubAt
		row.LastSubAt = &t
	}
	if st.UA != "" {
		v := st.UA
		row.LastSubUA = &v
	}
	if st.IP != "" {
		v := st.IP
		row.LastSubIP = &v
	}
	if st.Location != "" {
		v := st.Location
		row.LastSubLocation = &v
	}
	if st.Format != "" {
		v := st.Format
		row.LastSubFormat = &v
	}
	if st.Software != "" {
		v := st.Software
		row.LastSubSoftware = &v
	}
	if err := db.Save(&row).Error; err != nil {
		log.Printf("activity: 保存活动记录失败 user=%d: %v", userID, err)
	}
}

func persistNodeReport(st NodeUserStat, at time.Time) {
	db := database.GetDB()
	if db == nil || st.UserID == 0 {
		return
	}
	var row models.UserUsageState
	if err := db.Where("user_id = ?", st.UserID).FirstOrInit(&row).Error; err != nil {
		log.Printf("activity: 读取节点上报记录失败 user=%d: %v", st.UserID, err)
		return
	}
	row.UserID = st.UserID
	row.OnlineConns = st.Conns
	row.LastNodeReportAt = &at
	// 流量按增量累加（同一用户并发上报时极小概率丢一次增量，心跳周期 30-60s，可接受）
	row.NodeTrafficUp += st.Up
	row.NodeTrafficDown += st.Down
	if err := db.Save(&row).Error; err != nil {
		log.Printf("activity: 保存节点上报失败 user=%d: %v", st.UserID, err)
	}
}

func loadPersisted(userID uint) (*models.UserUsageState, error) {
	db := database.GetDB()
	if db == nil {
		return nil, nil
	}
	var row models.UserUsageState
	if err := db.Where("user_id = ?", userID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func applyPersisted(st *State, row *models.UserUsageState, now time.Time) {
	if row == nil {
		return
	}
	if row.LastSubAt != nil {
		st.LastSubAt = *row.LastSubAt
		st.LastActiveAt = *row.LastSubAt
		st.SubFetchCount = row.SubFetchCount
		age := now.Sub(*row.LastSubAt)
		switch {
		case age <= ActiveWindow:
			st.State = StateActive
		case age <= IdleWindow:
			st.State = StateIdle
		}
	}
	if row.LastSubUA != nil {
		st.UA = *row.LastSubUA
	}
	if row.LastSubIP != nil {
		st.IP = *row.LastSubIP
	}
	if row.LastSubLocation != nil {
		st.Location = *row.LastSubLocation
	}
	if row.LastSubFormat != nil {
		st.Format = *row.LastSubFormat
	}
	if row.LastSubSoftware != nil {
		st.Software = *row.LastSubSoftware
	}
	if row.LastNodeReportAt != nil && now.Sub(*row.LastNodeReportAt) <= OnlineTTL {
		st.State = StateOnline
		st.OnlineConns = row.OnlineConns
		st.LastNodeReportAt = *row.LastNodeReportAt
	}
}
