module gitlab.com/elixxir/gateway

go 1.13

require (
	github.com/denisenkom/go-mssqldb v0.0.0-20200910202707-1e08a3fab204 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/jinzhu/gorm v1.9.16
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/lib/pq v1.8.0 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.4-0.20201113184530-f02d398215b5
	gitlab.com/elixxir/crypto v0.0.5-0.20201110193609-6b5e881867b4
	gitlab.com/elixxir/primitives v0.0.2
	gitlab.com/xx_network/comms v0.0.4-0.20201110022115-4a6171cad07d
	gitlab.com/xx_network/crypto v0.0.4
	gitlab.com/xx_network/primitives v0.0.2
	gopkg.in/ini.v1 v1.62.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
