// Create package in v.1.0.0
// Same as entities, struct and method in domain package will used in all layer.
// srvcheck.go is file that define model as struct and abstract method of model as interface.
// Also, it declare usecase interface used as business layer.

// srvcheck domain is managing the state of the service (elasticsearch, swarm, consul, etc ...) periodically

// All model struct and interface is about service check domain
// Most importantly, it only defines and does not implement interfaces.

package domain

import (
	"strings"
	"time"
)

// serviceCheckHistoryComponent is basic model using by embedded in every model struct about service check history
type serviceCheckHistoryComponent struct {
	// private field in below, these fields have fixed value so always set in FillPrivateComponent method
	// Agent specifies name of service that created this model
	agent string

	// version specifies health checker version when this model was created
	version string

	// Timestamp specifies the time when this model was created.
	timestamp time.Time

	// Domain specifies domain about right this package, srvcheck
	domain string

	// _type specifies detail service type in service check domain (Ex, elasticsearch, swarm, consul, etc ...)
	_type string

	// ---

	// public field in below, these fields don't have fixed value so set in another package from custom user
	// UUID specifies Universally unique identifier in each of service check process
	UUID string

	// ProcessLevel specifies about how level to handle service check process.
	ProcessLevel srvcheckProcessLevel

	// Message specifies additional description of result about service check process.
	Message string

	// Error specifies error message if health check's been handled abnormally.
	Error error

	// ---

	// field in below is about alarm result and is private so call SetAlarmResult method to set this field value
	// Alerted specifies if alert result or status in while handling service check process.
	alerted bool

	// alarmText specifies alarm text sent in service check process.
	alarmText string

	// alarmTime specifies time when this service check sent alarm.
	alarmTime time.Time

	// alarmErr specifies Error occurred when sending alarm.
	alarmErr error
}

// serviceCheckHistoryRepositoryComponent is basic interface using by embedded in every repository about service check history
type serviceCheckHistoryRepositoryComponent interface {
	// Migrate method build environment for storage in stores such as Mysql or Elasticsearch, etc.
	Migrate() error
}

// FillComponent fill field of serviceCheckHistoryComponent if is empty
func (sch *serviceCheckHistoryComponent) FillPrivateComponent() {
	sch.version = version
	sch.agent = "sms-health-check"
	sch.domain = "srvcheck"
	sch._type = "None"
	sch.timestamp = time.Now()
}

// DottedMapWithPrefix convert serviceCheckHistoryComponent to dotted map and return that
// all key value of Map start with prefix received from parameter
func (sch *serviceCheckHistoryComponent) DottedMapWithPrefix(prefix string) (m map[string]interface{}) {
	if prefix != "" {
		prefix += "."
	}

	m = map[string]interface{}{}

	// setting private field value in dotted map
	m[prefix + "version"] = sch.version
	m[prefix + "agent"] = sch.agent
	m[prefix + "@timestamp"] = sch.timestamp
	m[prefix + "domain"] = sch.domain
	m[prefix + "type"] = sch._type

	// setting public field value in dotted map
	m[prefix + "uuid"] = sch.UUID
	m[prefix + "process_level"] = sch.ProcessLevel.String()
	m[prefix + "message"] = sch.Message
	if sch.Error == nil {
		m[prefix + "error"] = nil
	} else {
		m[prefix + "error"] = sch.Error.Error()
	}

	// setting alarm result field value in dotted map
	m[prefix + "alerted"] = sch.alerted
	m[prefix + "alarm_text"] = sch.alarmText
	m[prefix + "alarm_time"] = sch.alarmTime
	m[prefix + "alarm_error"] = sch.alarmErr

	return
}

// SetAlarmResult set field value about alarm result with parameter
func (sch *serviceCheckHistoryComponent) SetAlarmResult(t time.Time, text string, err error) {
	sch.alerted = true
	sch.alarmTime = t
	sch.alarmText = text
	sch.alarmErr = err
}

// SetError method set Message & Error field with err get from param
func (sch *serviceCheckHistoryComponent) SetError(err error) {
	sch.Message = err.Error()
	sch.Error = err
}

// srvcheckProcessLevel is string custom type used for representing service check process level
type srvcheckProcessLevel []string

// Set method overwrite srvcheckProcessLevel slice to level received from parameter
func (pl *srvcheckProcessLevel) Set(level string) {
	*pl = srvcheckProcessLevel{level}
}

// Append method append srvcheckProcessLevel slice with level received from parameter
func (pl *srvcheckProcessLevel) Append(level string) {
	for _, l := range *pl {
		if l == level {
			return
		}
	}
	*pl = append(*pl, level)
}

// String method return string which join srvcheckProcessLevel slice to string with " | "
func (pl *srvcheckProcessLevel) String() string {
	return strings.Join(*pl, " | ")
}
