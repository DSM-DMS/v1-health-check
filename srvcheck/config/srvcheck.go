// Create package in v.1.0.0.
// config package contains App global variable with config value about srvcheck from environment variable or config file
// App return field value from method having same name with that field name

// config.go is file that define srvcheckConfig type which is type of App
// Also, App implement various config interface each of package in srvcheck domain by declaring method

package config

import (
	"github.com/inhies/go-bytesize"
	"github.com/spf13/viper"
	"strings"
	"time"
)

// App is the application config about srvcheck domain
var App *srvcheckConfig

// srvcheckConfig having config value and implement various interface about Config by declaring method
type srvcheckConfig struct {
	// fields about index information in elasticsearch (implement esRepositoryComponentConfig)
	// indexName represent name of elasticsearch index including srvcheck history document
	indexName *string

	// indexShardNum represent shard number of elasticsearch index storing srvcheck history document
	indexShardNum *int

	// indexReplicaNum represent replica number of elasticsearch index to replace index when node become unable
	indexReplicaNum *int

	// ---

	// fields using in elasticsearch health checking (implement elasticsearchCheckUsecaseConfig)
	// maximumShardsNumber represent maximum shards number of elasticsearch target cluster
	maximumShardsNumber *int

	// jaegerIndexPattern represent jaeger index pattern to deliver to elasticsearch agency
	jaegerIndexPattern *string

	// jaegerIndexMinLifeCycle represent minimum life cycle of jaeger index in elasticsearch
	jaegerIndexMinLifeCycle *time.Duration

	// ---

	// fields using in swarmpit health checking (implement swarmpitCheckUsecaseConfig)
	// swarmpitAppServiceName represent swarmpit app service name in docker swarm
	swarmpitAppServiceName *string

	// swarmpitAppMaxMemoryUsage represent maximum memory usage of swarmpit app container
	swarmpitAppMaxMemoryUsage *bytesize.ByteSize

	// ---

	// fields using in consul health checking (implement consulCheckUsecaseConfig)
	// checkTargetServices represent check target services in check usecase
	checkTargetServices *[]string

	// consulServiceNameSpace represent consul service name space saved in consul
	consulServiceNameSpace *string

	// dockerServiceNameSpace represent docker service name space saved in docker
	dockerServiceNameSpace *string

	// ConnCheckPingTimeOut represent connection ping time out when check connection
	connCheckPingTimeOut *time.Duration

	// ---

	// fields using in main function to inject delivery layer (not implement any interface)
	// esCheckDeliveryPingCycle represent elasticsearch check delivery ping cycle
	esCheckDeliveryPingCycle *time.Duration

	// swarmpitCheckDeliveryPingCycle represent swarmpit check delivery ping cycle
	swarmpitCheckDeliveryPingCycle *time.Duration

	// consulCheckDeliveryPingCycle represent consul check delivery ping cycle
	consulCheckDeliveryPingCycle *time.Duration
}

const (
	defaultIndexName       = "sms-service-check" // default const string for indexName
	defaultIndexShardNum   = 2                   // default const int for indexShardNum
	defaultIndexReplicaNum = 0                   // default const int for indexReplicaNum

	defaultMaximumShardsNumber     = 900             // default const int for MaximumShardsNumber
	defaultJaegerIndexMinLifeCycle = time.Hour * 720 // default const duration for JaegerIndexMinLifeCycle
	defaultJaegerIndexPattern      = "jaeger-*"      // default const string for JaegerIndexRegexp

	defaultSwarmpitAppServiceName    = "swarmpit_app"    // default const string for swarmpitAppServiceName
	defaultSwarmpitAppMaxMemoryUsage = bytesize.MB * 600 // default const bytesize for swarmpitAppMaxMemoryUsage

	defaultCheckTargetServices    = "announcement,auth,club,outing,schedule" // default const string for checkTargetServices
	defaultConsulServiceNameSpace = "DMS.SMS.v1.service."                    // default const string for consulServiceNameSpace
	defaultDockerServiceNameSpace = "DSM_SMS_service-"                       // default const string for dockerServiceNameSpace
	defaultConnCheckPingTimeOut   = time.Second * 5                          // default const duration for connCheckPingTimeOut

	defaultESCheckDeliveryPingCycle       = time.Hour * 12  // default const Duration for esCheckDeliveryPingCycle
	defaultSwarmpitCheckDeliveryPingCycle = time.Hour * 6   // default const Duration for swarmpitCheckDeliveryPingCycle
	defaultConsulCheckDeliveryPingCycle   = time.Minute * 1 // default const Duration for consulCheckDeliveryPingCycle
)

// implement IndexName method of esRepositoryComponentConfig interface
func (sc *srvcheckConfig) IndexName() string {
	var key = "srvcheck.repository.elasticsearch.index.name"
	if sc.indexName == nil {
		if _, ok := viper.Get(key).(string); !ok {
			viper.Set(key, defaultIndexName)
		}
		sc.indexName = _string(viper.GetString(key))
	}
	return *sc.indexName
}

// implement IndexShardNum method of esRepositoryComponentConfig interface
func (sc *srvcheckConfig) IndexShardNum() int {
	var key = "srvcheck.repository.elasticsearch.index.shardNum"
	if sc.indexShardNum == nil {
		if _, ok := viper.Get(key).(int); !ok {
			viper.Set(key, defaultIndexShardNum)
		}
		sc.indexShardNum = _int(viper.GetInt(key))
	}
	return *sc.indexShardNum
}

// implement IndexReplicaNum method of esRepositoryComponentConfig interface
func (sc *srvcheckConfig) IndexReplicaNum() int {
	var key = "srvcheck.repository.elasticsearch.index.replicaNum"
	if sc.indexReplicaNum == nil {
		if _, ok := viper.Get(key).(int); !ok {
			viper.Set(key, defaultIndexReplicaNum)
		}
		sc.indexReplicaNum = _int(viper.GetInt(key))
	}
	return *sc.indexReplicaNum
}

// implement MaximumShardsNumber method of elasticsearchCheckUsecaseConfig interface
func (sc *srvcheckConfig) MaximumShardsNumber() int {
	var key = "srvcheck.elasticsearch.maximumShardsNumber"
	if sc.maximumShardsNumber == nil {
		if _, ok := viper.Get(key).(int); !ok {
			viper.Set(key, defaultMaximumShardsNumber)
		}
		sc.maximumShardsNumber = _int(viper.GetInt(key))
	}
	return *sc.maximumShardsNumber
}

// implement JaegerIndexMinLifeCycle method of elasticsearchCheckUsecaseConfig interface
func (sc *srvcheckConfig) JaegerIndexMinLifeCycle() time.Duration {
	var key = "srvcheck.elasticsearch.jaegerIndexMinLifeCycle"
	if sc.jaegerIndexMinLifeCycle != nil {
		return *sc.jaegerIndexMinLifeCycle
	}

	d, err := time.ParseDuration(viper.GetString(key))
	if err != nil {
		viper.Set(key, defaultJaegerIndexMinLifeCycle.String())
		d = defaultJaegerIndexMinLifeCycle
	}

	sc.jaegerIndexMinLifeCycle = &d
	return *sc.jaegerIndexMinLifeCycle
}

// implement JaegerIndexPattern method of elasticsearchCheckUsecaseConfig interface
func (sc *srvcheckConfig) JaegerIndexPattern() string {
	var key = "srvcheck.elasticsearch.jaegerIndexPattern"
	if sc.jaegerIndexPattern == nil {
		if _, ok := viper.Get(key).(string); !ok {
			viper.Set(key, defaultJaegerIndexPattern)
		}
		sc.jaegerIndexPattern = _string(viper.GetString(key))
	}
	return *sc.jaegerIndexPattern
}

// implement SwarmpitAppServiceName method of swarmpitCheckUsecaseConfig interface
func (sc *srvcheckConfig) SwarmpitAppServiceName() string {
	var key = "srvcheck.swarmpit.swarmpitAppServiceName"
	if sc.swarmpitAppServiceName == nil {
		if _, ok := viper.Get(key).(string); !ok {
			viper.Set(key, defaultSwarmpitAppServiceName)
		}
		sc.swarmpitAppServiceName = _string(viper.GetString(key))
	}
	return *sc.swarmpitAppServiceName
}

// implement DiskMinCapacity method of swarmpitCheckUsecaseConfig interface
func (sc *srvcheckConfig) SwarmpitAppMaxMemoryUsage() bytesize.ByteSize {
	var key = "srvcheck.swarmpit.swarmpitAppMaxMemoryUsage"
	if sc.swarmpitAppMaxMemoryUsage != nil {
		return *sc.swarmpitAppMaxMemoryUsage
	}

	size, err := bytesize.Parse(viper.GetString(key))
	if err != nil {
		viper.Set(key, defaultSwarmpitAppMaxMemoryUsage.String())
		size = defaultSwarmpitAppMaxMemoryUsage
	}

	sc.swarmpitAppMaxMemoryUsage = &size
	return *sc.swarmpitAppMaxMemoryUsage
}

// implement CheckTargetServices method of consulCheckUsecaseConfig interface
func (sc *srvcheckConfig) CheckTargetServices() []string {
	var key = "srvcheck.consul.checkTargetServices"
	if sc.checkTargetServices == nil {
		if _, ok := viper.Get(key).(string); !ok {
			viper.Set(key, defaultCheckTargetServices)
		}
		sep := strings.Split(viper.GetString(key), ",")
		sc.checkTargetServices = &sep
	}
	return *sc.checkTargetServices
}

// implement ConsulServiceNameSpace method of consulCheckUsecaseConfig interface
func (sc *srvcheckConfig) ConsulServiceNameSpace() string {
	var key = "srvcheck.consul.consulServiceNameSpace"
	if sc.consulServiceNameSpace == nil {
		if _, ok := viper.Get(key).(string); !ok {
			viper.Set(key, defaultConsulServiceNameSpace)
		}
		sc.consulServiceNameSpace = _string(viper.GetString(key))
	}
	return *sc.consulServiceNameSpace
}

// implement DockerServiceNameSpace method of consulCheckUsecaseConfig interface
func (sc *srvcheckConfig) DockerServiceNameSpace() string {
	var key = "srvcheck.consul.dockerServiceNameSpace"
	if sc.dockerServiceNameSpace == nil {
		if _, ok := viper.Get(key).(string); !ok {
			viper.Set(key, defaultDockerServiceNameSpace)
		}
		sc.dockerServiceNameSpace = _string(viper.GetString(key))
	}
	return *sc.dockerServiceNameSpace
}

// implement ConnCheckPingTimeOut method of consulCheckUsecaseConfig interface
func (sc *srvcheckConfig) ConnCheckPingTimeOut() time.Duration {
	var key = "srvcheck.consul.connCheckPingTimeOut"
	if sc.connCheckPingTimeOut != nil {
		return *sc.connCheckPingTimeOut
	}

	d, err := time.ParseDuration(viper.GetString(key))
	if err != nil {
		viper.Set(key, defaultConnCheckPingTimeOut.String())
		d = defaultConnCheckPingTimeOut
	}

	sc.connCheckPingTimeOut = &d
	return *sc.connCheckPingTimeOut
}

// not implement any interface, just using in main function for delivery layer injection
func (sc *srvcheckConfig) ESCheckDeliveryPingCycle() time.Duration {
	var key = "srvcheck.delivery.channel.pingCycle.elasticsearchCheck"
	if sc.esCheckDeliveryPingCycle != nil {
		return *sc.esCheckDeliveryPingCycle
	}

	d, err := time.ParseDuration(viper.GetString(key))
	if err != nil {
		viper.Set(key, defaultESCheckDeliveryPingCycle.String())
		d = defaultESCheckDeliveryPingCycle
	}

	sc.esCheckDeliveryPingCycle = &d
	return *sc.esCheckDeliveryPingCycle
}

// not implement any interface, just using in main function for delivery layer injection
func (sc *srvcheckConfig) SwarmpitCheckDeliveryPingCycle() time.Duration {
	var key = "srvcheck.delivery.channel.pingCycle.swarmpitCheck"
	if sc.swarmpitCheckDeliveryPingCycle != nil {
		return *sc.swarmpitCheckDeliveryPingCycle
	}

	d, err := time.ParseDuration(viper.GetString(key))
	if err != nil {
		viper.Set(key, defaultSwarmpitCheckDeliveryPingCycle.String())
		d = defaultSwarmpitCheckDeliveryPingCycle
	}

	sc.swarmpitCheckDeliveryPingCycle = &d
	return *sc.swarmpitCheckDeliveryPingCycle
}

// not implement any interface, just using in main function for delivery layer injection
func (sc *srvcheckConfig) ConsulCheckDeliveryPingCycle() time.Duration {
	var key = "srvcheck.delivery.channel.pingCycle.consulCheck"
	if sc.consulCheckDeliveryPingCycle != nil {
		return *sc.consulCheckDeliveryPingCycle
	}

	d, err := time.ParseDuration(viper.GetString(key))
	if err != nil {
		viper.Set(key, defaultConsulCheckDeliveryPingCycle.String())
		d = defaultConsulCheckDeliveryPingCycle
	}

	sc.consulCheckDeliveryPingCycle = &d
	return *sc.consulCheckDeliveryPingCycle
}

// init function initialize App global variable
func init() {
	App = &srvcheckConfig{}
}

// function returns pointer variable generated from parameter
func _string(s string) *string { return &s }
func _int(i int) *int {return &i}
