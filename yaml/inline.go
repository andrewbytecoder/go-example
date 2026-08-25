package yaml

type ImageConfig struct {
	Repository      string `yaml:"repository"`
	Tag             string `yaml:"tag"`
	ImagePullPolicy string `yaml:"imagePullPolicy"`
}

// ImageConfigs 镜像配置
type ImageConfigs struct {
	App      ImageConfig `yaml:"app"`
	StartSeq ImageConfig `yaml:"startSeq"`
	Sidecar  ImageConfig `yaml:"sidecar"`
}

// GImages 全局镜像配置
// 如果使用yaml.v3进行yaml解析，当结构体进行内嵌组合的时候，需要显式指定yaml.v3的解析规则
// 必须指定 yaml:",inline" 才能正常运行
type GImages struct {
	Prefix          string `yaml:"prefix"`
	ImagePrefix     string `yaml:"imagePrefix"`
	ImagePullPolicy string `yaml:"imagePullPolicy"`
	// yaml.v3 中匿名字段不会自动内联（与 yaml.v2 不同），必须显式指定 ,inline
	// 否则 global.images 下的 app/startSeq/sidecar 全部解析不到
	ImageConfigs `yaml:",inline"`
}
