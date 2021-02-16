module gitlab.com/elixxir/gateway

go 1.13

require (
	github.com/golang/protobuf v1.4.3
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/jinzhu/gorm v1.9.16
	github.com/lib/pq v1.9.0 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/bloomfilter v0.0.0-20210120224144-ac046983a17a
	gitlab.com/elixxir/comms v0.0.4-0.20210216174714-dc3e5c456e20
	gitlab.com/elixxir/crypto v0.0.7-0.20210216174551-f806f79610eb
	gitlab.com/elixxir/primitives v0.0.3-0.20210216174458-2a23825c1eb1
	gitlab.com/xx_network/comms v0.0.4-0.20210216174438-0790d1f1f225
	gitlab.com/xx_network/crypto v0.0.5-0.20210216174356-e81e1ddf8fb7
	gitlab.com/xx_network/primitives v0.0.4-0.20210215192713-e32335847d4f
	gitlab.com/xx_network/ring v0.0.3-0.20201120004140-b0e268db06d1 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sys v0.0.0-20210105210732-16f7687f5001 // indirect
	google.golang.org/genproto v0.0.0-20210105202744-fe13368bc0e1 // indirect
	google.golang.org/grpc v1.34.0 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gorm.io/driver/postgres v1.0.7
	gorm.io/gorm v1.20.12
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
