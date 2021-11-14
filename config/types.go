package config

type Config struct {
	Addr       string         `yaml:"listen_addr"`
	Processing ProcessingType `yaml:"processing"`
	Prometheus PrometheusType `yaml:"prometheus"`
}

type ProcessingType struct {
	DefaultResourceQuotaName    string                 `yaml:"default_resource_quota_name"`
	InfraFee                    string                 `yaml:"infra_fee"`
	InfraCustomers              []string               `yaml:"infra_customers"`
	CpuOversubscription         string                 `yaml:"cpu_oversubscription"`
	BusinessAnnotationFieldName string                 `yaml:"business_annotation_field_name"`
	DefaultLimitRange           map[string]interface{} `yaml:"default_limitrange"`
}

type PrometheusType struct {
	Address string `yaml:"addr"`
}
