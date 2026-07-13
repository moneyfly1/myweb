package config_update

type TransportOpts struct {
	WSOpts *WSOpts `yaml:"ws-opts,omitempty" json:"ws-opts,omitempty"`

	GRPCOpts *GRPCOpts `yaml:"grpc-opts,omitempty" json:"grpc-opts,omitempty"`

	H2Opts *H2Opts `yaml:"h2-opts,omitempty" json:"h2-opts,omitempty"`

	SNI               string `yaml:"servername,omitempty" json:"servername,omitempty"`
	SkipCertVerify    bool   `yaml:"skip-cert-verify,omitempty" json:"skip-cert-verify,omitempty"`
	ClientFingerprint string `yaml:"client-fingerprint,omitempty" json:"client-fingerprint,omitempty"`

	RealityOpts *RealityOpts `yaml:"reality-opts,omitempty" json:"reality-opts,omitempty"`

	Other map[string]interface{} `yaml:",inline" json:",inline"`
}

type WSOpts struct {
	Path             string            `yaml:"path,omitempty" json:"path,omitempty"`
	Headers          map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	V2rayHTTPUpgrade bool              `yaml:"v2ray-http-upgrade,omitempty" json:"v2ray-http-upgrade,omitempty"`
}

type GRPCOpts struct {
	GRPCServiceName string `yaml:"grpc-service-name,omitempty" json:"grpc-service-name,omitempty"`
}

type H2Opts struct {
	Path string   `yaml:"path,omitempty" json:"path,omitempty"`
	Host []string `yaml:"host,omitempty" json:"host,omitempty"`
}

type RealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty" json:"public-key,omitempty"`
	ShortID   string `yaml:"short-id,omitempty" json:"short-id,omitempty"`
	PQV       string `yaml:"pqv,omitempty" json:"pqv,omitempty"`
}

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

	for k, v := range t.Other {
		result[k] = v
	}

	return result
}
