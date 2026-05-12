package payment

import "testing"

func TestBuildEpayOrderQueryURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mapi",
			in:   "https://pay.example.com/mapi.php",
			want: "https://pay.example.com/api.php?act=order",
		},
		{
			name: "submit with existing act",
			in:   "https://pay.example.com/submit.php?act=pay",
			want: "https://pay.example.com/api.php?act=order",
		},
		{
			name: "gateway root",
			in:   "https://pay.example.com",
			want: "https://pay.example.com/api.php?act=order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildEpayOrderQueryURL(tt.in); got != tt.want {
				t.Fatalf("buildEpayOrderQueryURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseEpayQueryResponse(t *testing.T) {
	raw, err := parseEpayQueryResponse(`{"code":1,"status":"TRADE_SUCCESS","out_trade_no":"ORD001","money":"12.30"}`)
	if err != nil {
		t.Fatalf("parse json response: %v", err)
	}
	result := normalizeEpayQueryResult("test", raw)
	if !result.IsPaid() || result.OutTradeNo != "ORD001" || result.Amount != "12.30" {
		t.Fatalf("unexpected json result: %+v", result)
	}

	raw, err = parseEpayQueryResponse("code=1&status=1&out_trade_no=RCH001&money=8.00")
	if err != nil {
		t.Fatalf("parse form response: %v", err)
	}
	result = normalizeEpayQueryResult("test", raw)
	if !result.IsPaid() || result.OutTradeNo != "RCH001" || result.Amount != "8.00" {
		t.Fatalf("unexpected form result: %+v", result)
	}
}

func TestWechatXMLQueryParse(t *testing.T) {
	raw, err := parseWechatXMLParams([]byte(`<xml>
		<return_code><![CDATA[SUCCESS]]></return_code>
		<result_code><![CDATA[SUCCESS]]></result_code>
		<trade_state><![CDATA[SUCCESS]]></trade_state>
		<out_trade_no><![CDATA[ORD001]]></out_trade_no>
		<transaction_id><![CDATA[WX001]]></transaction_id>
		<total_fee><![CDATA[1230]]></total_fee>
	</xml>`))
	if err != nil {
		t.Fatalf("parse wechat xml: %v", err)
	}
	result := &WechatQueryResult{
		ReturnCode:    raw["return_code"],
		ResultCode:    raw["result_code"],
		TradeState:    raw["trade_state"],
		OutTradeNo:    raw["out_trade_no"],
		TransactionID: raw["transaction_id"],
		TotalFee:      raw["total_fee"],
		Raw:           raw,
	}
	if !result.IsPaid() || result.OutTradeNo != "ORD001" || result.TotalFee != "1230" {
		t.Fatalf("unexpected wechat result: %+v", result)
	}
}
