package config_update

// TransportOpts 传输选项结构体（替代 map[string]interface{}）
type TransportOpts struct {
	// WebSocket 选项
	WSOpts *WSOpts `yaml:"ws-opts,omitempty" json:"ws-opts,omitempty"`
	
	// gRPC 选项
	GRPCOpts *GRPCOpts `yaml:"grpc-opts,omitempty" json:"grpc-opts,omitempty"`
	
	// HTTP/2 选项
	H2Opts *H2Opts `yaml:"h2-opts,omitempty" json:"h2-opts,omitempty"`
	
	// TLS 选项
	SNI              string `yaml:"servername,omitempty" json:"servername,omitempty"`
	SkipCertVerify   bool   `yaml:"skip-cert-verify,omitempty" json:"skip-cert-verify,omitempty"`
	ClientFingerprint string `yaml:"client-fingerprint,omitempty" json:"client-fingerprint,omitempty"`
	
	// Reality 选项
	RealityOpts *RealityOpts `yaml:"reality-opts,omitempty" json:"reality-opts,omitempty"`
	
	// 其他选项（保留兼容性）
	Other map[string]interface{} `yaml:",inline" json:",inline"`
}

// WSOpts WebSocket 选项
type WSOpts struct {
	Path    string            `yaml:"path,omitempty" json:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	V2rayHTTPUpgrade bool     `yaml:"v2ray-http-upgrade,omitempty" json:"v2ray-http-upgrade,omitempty"`
}

// GRPCOpts gRPC 选项
type GRPCOpts struct {
	GRPCServiceName string `yaml:"grpc-service-name,omitempty" json:"grpc-service-name,omitempty"`
}

// H2Opts HTTP/2 选项
type H2Opts struct {
	Path string   `yaml:"path,omitempty" json:"path,omitempty"`
	Host []string `yaml:"host,omitempty" json:"host,omitempty"`
}

// RealityOpts Reality 选项
type RealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty" json:"public-key,omitempty"`
	ShortID   string `yaml:"short-id,omitempty" json:"short-id,omitempty"`
	PQV       string `yaml:"pqv,omitempty" json:"pqv,omitempty"`
}

// ToMap 转换为 map[string]interface{}（用于向后兼容）
func (t *TransportOpts) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	
	if t.WSOpts != nil {
		result["ws-opts"] = map[string]interface{}{
			"path":    t.WSOpts.Path,
			"headers": t.WSOpts.Headers,
		}
		if t.WSOpts.V2rayHTTPUpgrade {
			result["ws-opts"].(map[string]interface{})["v2ray-http-upgrade"] = true
		}
	}
	
	if t.GRPCOpts != nil {
		result["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": t.GRPCOpts.GRPCServiceName,
		}
	}
	
	if t.H2Opts != nil {
		result["h2-opts"] = map[string]interface{}{
			"path": t.H2Opts.Path,
			"host": t.H2Opts.Host,
		}
	}
	
	if t.SNI != "" {
		result["servername"] = t.SNI
	}
	
	if t.SkipCertVerify {
		result["skip-cert-verify"] = true
	}
	
	if t.ClientFingerprint != "" {
		result["client-fingerprint"] = t.ClientFingerprint
	}
	
	if t.RealityOpts != nil {
		realityMap := make(map[string]interface{})
		if t.RealityOpts.PublicKey != "" {
			realityMap["public-key"] = t.RealityOpts.PublicKey
		}
		if t.RealityOpts.ShortID != "" {
			realityMap["short-id"] = t.RealityOpts.ShortID
		}
		if t.RealityOpts.PQV != "" {
			realityMap["pqv"] = t.RealityOpts.PQV
		}
		if len(realityMap) > 0 {
			result["reality-opts"] = realityMap
		}
	}
	
	// 合并其他选项
	for k, v := range t.Other {
		result[k] = v
	}
	
	return result
}

// FromMap 从 map[string]interface{} 创建 TransportOpts（用于向后兼容）
func TransportOptsFromMap(m map[string]interface{}) *TransportOpts {
	if m == nil {
		return nil
	}
	
	opts := &TransportOpts{
		Other: make(map[string]interface{}),
	}
	
	// 解析 ws-opts
	if wsOpts, ok := m["ws-opts"].(map[string]interface{}); ok {
		opts.WSOpts = &WSOpts{
			Headers: make(map[string]string),
		}
		if path, ok := wsOpts["path"].(string); ok {
			opts.WSOpts.Path = path
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if s, ok := v.(string); ok {
					opts.WSOpts.Headers[k] = s
				}
			}
		}
		if v2ray, ok := wsOpts["v2ray-http-upgrade"].(bool); ok && v2ray {
			opts.WSOpts.V2rayHTTPUpgrade = true
		}
	}
	
	// 解析 grpc-opts
	if grpcOpts, ok := m["grpc-opts"].(map[string]interface{}); ok {
		opts.GRPCOpts = &GRPCOpts{}
		if name, ok := grpcOpts["grpc-service-name"].(string); ok {
			opts.GRPCOpts.GRPCServiceName = name
		}
	}
	
	// 解析 h2-opts
	if h2Opts, ok := m["h2-opts"].(map[string]interface{}); ok {
		opts.H2Opts = &H2Opts{}
		if path, ok := h2Opts["path"].(string); ok {
			opts.H2Opts.Path = path
		}
		if host, ok := h2Opts["host"].([]interface{}); ok {
			opts.H2Opts.Host = make([]string, 0, len(host))
			for _, h := range host {
				if s, ok := h.(string); ok {
					opts.H2Opts.Host = append(opts.H2Opts.Host, s)
				}
			}
		}
	}
	
	// 解析 TLS 选项
	if sni, ok := m["servername"].(string); ok {
		opts.SNI = sni
	}
	if skip, ok := m["skip-cert-verify"].(bool); ok {
		opts.SkipCertVerify = skip
	}
	if fp, ok := m["client-fingerprint"].(string); ok {
		opts.ClientFingerprint = fp
	}
	
	// 解析 reality-opts
	if realityOpts, ok := m["reality-opts"].(map[string]interface{}); ok {
		opts.RealityOpts = &RealityOpts{}
		if pk, ok := realityOpts["public-key"].(string); ok {
			opts.RealityOpts.PublicKey = pk
		}
		if sid, ok := realityOpts["short-id"].(string); ok {
			opts.RealityOpts.ShortID = sid
		}
		if pqv, ok := realityOpts["pqv"].(string); ok {
			opts.RealityOpts.PQV = pqv
		}
	}
	
	// 其他选项
	for k, v := range m {
		switch k {
		case "ws-opts", "grpc-opts", "h2-opts", "servername", "skip-cert-verify", "client-fingerprint", "reality-opts":
			// 已处理
		default:
			opts.Other[k] = v
		}
	}
	
	return opts
}

