package migrate

type Config struct {
	SchemaName string
	TableName  string
}

func EffectiveConfig(d Dialect, c *Config) *Config {
	if d == nil {
		return c
	}
	ret := d.DefaultConfig()
	if c == nil {
		return ret
	}
	if c.SchemaName != "" {
		ret.SchemaName = c.SchemaName
	}
	if c.TableName != "" {
		ret.TableName = c.TableName
	}
	return ret
}
