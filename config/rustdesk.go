package config

import (
	"os"
)

const (
	DefaultIdServerPort    = 21116
	DefaultRelayServerPort = 21117
)

type Rustdesk struct {
	IdServer             string `mapstructure:"id-server"`
	IdServerPort         int    `mapstructure:"-"`
	RelayServer          string `mapstructure:"relay-server"`
	RelayServerPort      int    `mapstructure:"-"`
	ApiServer            string `mapstructure:"api-server"`
	HbbsInternalUrl      string `mapstructure:"hbbs-internal-url"`
	HbbsInternalKey      string `mapstructure:"hbbs-internal-key"`
	AccessControlEnabled bool   `mapstructure:"access-control-enabled"`
	Key                  string `mapstructure:"key"`
	KeyFile              string `mapstructure:"key-file"`
	Personal             int    `mapstructure:"personal"`
	//webclient-magic-queryonline
	WebclientMagicQueryonline int    `mapstructure:"webclient-magic-queryonline"`
	WsHost                    string `mapstructure:"ws-host"`
}

func (rd *Rustdesk) LoadKeyFile() {
	// Prefer an explicitly configured key file. Container deployments share the
	// hbbs public key through this path, while the bundled config may contain a
	// development key that must not override the mounted server identity.
	if rd.KeyFile != "" {
		b, err := os.ReadFile(rd.KeyFile)
		if err == nil && len(b) > 0 {
			rd.Key = string(b)
		}
	}
}
