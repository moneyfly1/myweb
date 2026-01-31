package payment

import (
	"strings"

	"cboard-go/internal/models"
)

type YipayPlatformAdapter interface {
	GetPlatformName() string
	GetAPIURL(gatewayURL string) string
	GetSubmitURL(gatewayURL string) string
	NormalizeResponse(resp map[string]interface{}) *YipayResponse
	GetResponseFields() YipayResponseFields
	SupportsSignatureType(signType string) bool
}

type YipayResponseFields struct {
	Code      string
	Msg       string
	TradeNo   string
	PayURL    string
	QRCode    string
	URLScheme string
	OID       string
}

type StandardYipayAdapter struct{}

func (a *StandardYipayAdapter) GetPlatformName() string {
	return "standard"
}

func (a *StandardYipayAdapter) GetAPIURL(gatewayURL string) string {
	gatewayURL = strings.TrimSuffix(gatewayURL, "/")
	if strings.HasSuffix(gatewayURL, "/mapi.php") {
		return gatewayURL
	}
	return gatewayURL + "/mapi.php"
}

func (a *StandardYipayAdapter) GetSubmitURL(gatewayURL string) string {
	gatewayURL = strings.TrimSuffix(gatewayURL, "/")
	return gatewayURL + "/submit.php"
}

func (a *StandardYipayAdapter) GetResponseFields() YipayResponseFields {
	return YipayResponseFields{
		Code:      "code",
		Msg:       "msg",
		TradeNo:   "trade_no",
		PayURL:    "payurl",
		QRCode:    "qrcode",
		URLScheme: "urlscheme",
		OID:       "o_id",
	}
}

func (a *StandardYipayAdapter) NormalizeResponse(resp map[string]interface{}) *YipayResponse {
	fields := a.GetResponseFields()
	result := &YipayResponse{}

	if code, ok := resp[fields.Code].(float64); ok {
		result.Code = int(code)
	} else if code, ok := resp[fields.Code].(int); ok {
		result.Code = code
	}

	if msg, ok := resp[fields.Msg].(string); ok {
		result.Msg = msg
	}

	if tradeNo, ok := resp[fields.TradeNo].(string); ok {
		result.TradeNo = tradeNo
	} else if oid, ok := resp[fields.OID].(string); ok {
		result.TradeNo = oid
	}

	if payURL, ok := resp[fields.PayURL].(string); ok {
		result.PayURL = payURL
	}

	if qrcode, ok := resp[fields.QRCode].(string); ok {
		result.QRCode = qrcode
	}

	if urlscheme, ok := resp[fields.URLScheme].(string); ok {
		result.URLScheme = urlscheme
	}

	return result
}

func (a *StandardYipayAdapter) SupportsSignatureType(signType string) bool {
	return signType == "MD5" || signType == "RSA" || signType == "MD5+RSA"
}

type YipayPlatformDetector struct {
	adapters map[string]YipayPlatformAdapter
}

func NewYipayPlatformDetector() *YipayPlatformDetector {
	detector := &YipayPlatformDetector{
		adapters: make(map[string]YipayPlatformAdapter),
	}

	standardAdapter := &StandardYipayAdapter{}
	detector.adapters["standard"] = standardAdapter

	return detector
}

func (d *YipayPlatformDetector) DetectPlatform(config *models.PaymentConfig) (YipayPlatformAdapter, string) {
	configData := parseConfigData(config.ConfigJSON)
	if configData == nil {
		return d.adapters["standard"], "standard"
	}

	platformName := getConfigString(configData, "platform_name")
	if platformName != "" {
		if adapter, ok := d.adapters[platformName]; ok {
			return adapter, platformName
		}
	}

	apiURL := getConfigString(configData, "api_url")
	gatewayURL := getConfigString(configData, "gateway_url")

	detectedPlatform := d.detectByURL(apiURL, gatewayURL)
	if adapter, ok := d.adapters[detectedPlatform]; ok {
		return adapter, detectedPlatform
	}

	return d.adapters["standard"], "standard"
}

func (d *YipayPlatformDetector) detectByURL(apiURL, gatewayURL string) string {
	urlToCheck := apiURL
	if urlToCheck == "" {
		urlToCheck = gatewayURL
	}
	if urlToCheck == "" {
		return "standard"
	}

	urlLower := strings.ToLower(urlToCheck)

	if strings.Contains(urlLower, "fhymw.com") ||
		strings.Contains(urlLower, "yi-zhifu.cn") ||
		strings.Contains(urlLower, "ezfp.cn") ||
		strings.Contains(urlLower, "myzfw.com") ||
		strings.Contains(urlLower, "8-pay.cn") ||
		strings.Contains(urlLower, "epay.hehanwang.com") ||
		strings.Contains(urlLower, "wx8g.com") {
		return "standard"
	}

	return "standard"
}

func (d *YipayPlatformDetector) RegisterAdapter(name string, adapter YipayPlatformAdapter) {
	d.adapters[name] = adapter
}

func detectYipayPlatform(config *models.PaymentConfig) (YipayPlatformAdapter, string) {
	detector := NewYipayPlatformDetector()
	return detector.DetectPlatform(config)
}
