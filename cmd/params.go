///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains Params-related functionality

package cmd

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/publicAddress"
	"gitlab.com/xx_network/comms/gossip"
	"gitlab.com/xx_network/primitives/rateLimiting"
)

type Params struct {
	NodeAddress      string `yaml:"cmixAddress"`
	Port             int
	PublicAddress    string // Gateway's public IP address (with port)
	ListeningAddress string // Gateway's local IP address (with port)
	CertPath         string
	KeyPath          string

	DbUsername string
	DbPassword string
	DbName     string
	DbAddress  string
	DbPort     string

	ServerCertPath        string `yaml:"cmixCertPath"`
	IDFPath               string
	PermissioningCertPath string `yaml:"schedulingCertPath"`

	rateLimitParams        *rateLimiting.MapParams
	messageRateLimitParams *rateLimiting.MapParams
	gossipFlags            gossip.ManagerFlags

	DevMode             bool
	DisableGossip       bool
	ignoreClientVersion bool

	cleanupInterval time.Duration
}

const (
	// Default time period for checking storage for stored items older
	// than the retention period value
	cleanupIntervalDefault = 5 * time.Minute
)

func InitParams(vip *viper.Viper) Params {
	if !validConfig {
		jww.FATAL.Panicf("Invalid Config File: %s", cfgFile)
	}

	// Print all config options
	jww.INFO.Printf("All config params: %+v", vip.AllKeys())

	certPath = viper.GetString("certPath")
	if certPath == "" {
		jww.FATAL.Panicf("Gateway.yaml certPath is required, path provided is empty.")
	}

	idfPath = viper.GetString("idfPath")
	if idfPath == "" {
		jww.FATAL.Panicf("Gateway.yaml idfPath is required, path provided is empty.")
	}
	keyPath = viper.GetString("keyPath")

	var nodeAddress string
	if viper.IsSet("cmixAddress") {
		nodeAddress = viper.GetString("cmixAddress")
	} else if viper.IsSet("nodeAddress") {
		nodeAddress = viper.GetString("nodeAddress")
	} else {
		jww.FATAL.Panicf("Gateway.yaml cmixAddress is required, address provided is empty.")
	}

	if viper.IsSet("schedulingCertPath") {
		permissioningCertPath = viper.GetString("schedulingCertPath")
	} else if viper.IsSet("permissioningCertPath") {
		permissioningCertPath = viper.GetString("permissioningCertPath")
	} else {
		jww.FATAL.Panicf("Gateway.yaml schedulingCertPath is required, path provided is empty.")
	}

	gwPort = viper.GetInt("port")
	if gwPort == 0 {
		jww.FATAL.Panicf("Gateway.yaml port is required, provided port is empty/not set.")
	}

	if viper.IsSet("cmixCertPath") {
		serverCertPath = viper.GetString("cmixCertPath")
	} else if viper.IsSet("serverCertPath") {
		serverCertPath = viper.GetString("serverCertPath")
	} else {
		jww.FATAL.Panicf("Gateway.yaml cmixCertPath is required, path provided is empty.")
	}

	// Get gateway's public IP or use the IP override
	overrideIP := viper.GetString("overridePublicIP")
	gwAddress, err := publicAddress.GetIpOverride(overrideIP, gwPort)
	if err != nil {
		jww.FATAL.Panicf("Failed to get public IP: %+v", err)
	}

	// Construct listening address
	listeningIP := viper.GetString("listeningAddress")
	if listeningIP == "" {
		listeningIP = "0.0.0.0"
	}
	listeningAddress := net.JoinHostPort(listeningIP, strconv.Itoa(gwPort))

	dbpass := viper.GetString("dbPassword")
	jww.INFO.Printf("config: %+v", viper.ConfigFileUsed())
	ps := fmt.Sprintf("Params: \n %+v", vip.AllSettings())
	ps = strings.ReplaceAll(ps,
		"dbpassword:"+dbpass,
		"dbpassword:[dbpass]")
	jww.INFO.Printf(ps)
	jww.INFO.Printf("Gateway port: %d", gwPort)
	jww.INFO.Printf("Gateway public IP: %s", gwAddress)
	jww.INFO.Printf("Gateway listening address: %s", listeningAddress)
	jww.INFO.Printf("Gateway node: %s", nodeAddress)

	// If the values aren't default, repopulate flag values with customized values
	// Otherwise use the default values
	gossipFlags := gossip.DefaultManagerFlags()
	if gossipFlags.BufferExpirationTime != bufferExpiration ||
		gossipFlags.MonitorThreadFrequency != monitorThreadFrequency {

		gossipFlags = gossip.ManagerFlags{
			BufferExpirationTime:   bufferExpiration,
			MonitorThreadFrequency: monitorThreadFrequency,
		}
	}

	// Construct the rate limiting params
	bucketMapParams := &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokens,
		LeakDuration: leakDuration,
		PollDuration: pollDuration,
		BucketMaxAge: bucketMaxAge,
	}

	messageLimitingParams := &rateLimiting.MapParams{
		Capacity:     1,
		LeakedTokens: 1,
		LeakDuration: 2 * time.Second,
		PollDuration: pollDuration,
		BucketMaxAge: bucketMaxAge,
	}

	// Time to periodically check for old objects in storage
	viper.SetDefault("cleanupInterval", cleanupIntervalDefault)
	cleanupInterval := viper.GetDuration("cleanupInterval")

	// Obtain database connection info
	rawAddr := viper.GetString("dbAddress")
	var addr, port string
	if rawAddr != "" {
		addr, port, err = net.SplitHostPort(rawAddr)
		if err != nil {
			jww.FATAL.Panicf("Unable to get database port from %s: %+v", rawAddr, err)
		}
	}

	return Params{
		Port:                   gwPort,
		PublicAddress:          gwAddress,
		ListeningAddress:       listeningAddress,
		NodeAddress:            nodeAddress,
		CertPath:               certPath,
		KeyPath:                keyPath,
		ServerCertPath:         serverCertPath,
		IDFPath:                idfPath,
		PermissioningCertPath:  permissioningCertPath,
		gossipFlags:            gossipFlags,
		rateLimitParams:        bucketMapParams,
		messageRateLimitParams: messageLimitingParams,
		DbName:                 viper.GetString("dbName"),
		DbUsername:             viper.GetString("dbUsername"),
		DbPassword:             viper.GetString("dbPassword"),
		DbAddress:              addr,
		DbPort:                 port,
		DevMode:                viper.GetBool("devMode"),
		DisableGossip:          viper.GetBool("disableGossip"),
		ignoreClientVersion:    viper.GetBool("ignoreClientVersion"),
		cleanupInterval:        cleanupInterval,
	}
}
