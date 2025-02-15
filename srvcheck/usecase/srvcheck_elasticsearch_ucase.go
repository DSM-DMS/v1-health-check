// Create file in v.1.0.0
// srvcheck_elasticsearch_ucase.go is file that define usecase implementation about elasticsearch check in srvcheck domain
// elasticsearch check usecase struct embed serviceCheckUsecaseComponent struct in ./srvcheck.go file

package usecase

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"sync"
	"time"

	"github.com/DMS-SMS/v1-health-check/domain"
)

// elasticsearchCheckStatus is type to int constant represent current elasticsearch check process status
type elasticsearchCheckStatus int
const (
	elasticsearchStatusHealthy    elasticsearchCheckStatus = iota // represent elasticsearch check status is healthy
	elasticsearchStatusRecovering                                 // represent it's recovering elasticsearch status now
	elasticsearchStatusUnhealthy                                  // represent elasticsearch check status is unhealthy
)

// elasticsearchCheckUsecase implement ElasticsearchCheckUsecase interface in domain and used in delivery layer
type elasticsearchCheckUsecase struct {
	// myCfg is used for getting elasticsearch check usecase config
	myCfg elasticsearchCheckUsecaseConfig

	// historyRepo is used for store elasticsearch check history and injected from outside
	historyRepo domain.ElasticsearchCheckHistoryRepository

	// slackChat is used for agent slack API about chatting
	slackChatAgency slackChatAgency

	// elasticsearchAgency is used as agency about elasticsearch API
	elasticsearchAgency elasticsearchAgency

	// status represent current process status of elasticsearch health check
	status elasticsearchCheckStatus

	// mutex help to prevent race condition when set status field value
	mutex sync.Mutex
}

// elasticsearchCheckUsecaseConfig is the config getter interface for elasticsearch check usecase
type elasticsearchCheckUsecaseConfig interface {
	// get common config method from embedding serviceCheckUsecaseComponentConfig
	serviceCheckUsecaseComponentConfig

	// MaximumShardsNumber method returns int represent maximum shards number
	MaximumShardsNumber() int

	// JaegerIndexPattern method returns string represent jaeger index pattern
	JaegerIndexPattern() string

	// JaegerIndexLifeCycle method returns duration represent jaeger index life cycle
	JaegerIndexMinLifeCycle() time.Duration
}

// elasticsearchAgency is interface that agent elasticsearch with HTTP API
type elasticsearchAgency interface {
	// GetClusterHealth return interface have various get method about cluster health inform
	GetClusterHealth() (cluster interface {
		ActivePrimaryShards() int     // get active primary shards number of cluster
		ActiveShards() int            // get active shards number of cluster
		UnassignedShards() int        // get unassigned shards number of cluster
		ActiveShardsPercent() float64 // get active shards percent of cluster
	}, err error)

	// GetIndicesWithRegexp return indices list with regexp pattern
	GetIndicesWithPatterns(patterns []string) (indices interface {
		SetMinLifeCycle(cycle time.Duration) // set min life cycle of index of indices
		IndexNames() []string                // get index name list of indices
	}, err error)

	// DeleteIndices method delete indices in list received from parameter
	DeleteIndices(indices []string) (err error)
}

// NewElasticsearchCheckUsecase function return elasticsearchCheckUseCase ptr instance after initializing
func NewElasticsearchCheckUsecase(
	cfg elasticsearchCheckUsecaseConfig,
	chr domain.ElasticsearchCheckHistoryRepository,
	sca slackChatAgency,
	ea elasticsearchAgency,
) domain.ElasticsearchCheckUseCase {
	return &elasticsearchCheckUsecase{
		// initialize field with parameter received from caller
		myCfg:               cfg,
		historyRepo:         chr,
		slackChatAgency:     sca,
		elasticsearchAgency: ea,

		// initialize field with default value
		status: elasticsearchStatusHealthy,
		mutex:  sync.Mutex{},
	}
}

// CheckElasticsearch check elasticsearch health with checkElasticsearch method & store check history in repository
// Implement CheckElasticsearch method of ElasticsearchCheckUseCase interface
func (ecu *elasticsearchCheckUsecase) CheckElasticsearch(ctx context.Context) (err error) {
	history := ecu.checkElasticsearch(ctx)

	if b, err := ecu.historyRepo.Store(history); err != nil {
		return errors.Wrapf(err, "failed to store elasticsearch check history, response: %s", string(b))
	}

	return
}

// method processed with below logic about elasticsearch health check according to current check status
// 0 : 정상적으로 인지된 상태 (상태 확인 수행)
// 0 -> 1 : Jaeger Index 삭제 실행 (Jaeger Index 삭제 알림 발행)
// 1 : Jaeger Index 삭제중 (상태 확인 수행 X)
// 1 -> 0 : Jaeger Index 삭제로 인해 상태 회복 완료 (상태 회복 알림 발행)
// 1 -> 2 : Jaeger Index 삭제를 해도 상태 회복 X (상태 회복 불가능 상태 알림 발행)
// 2 : 관리자가 직접 확인해야함 (상태 확인 수행 X)
// 2 -> 0 : 관리자 직접 상태 회복 완료 (상태 회복 알림 발행)
func (ecu *elasticsearchCheckUsecase) checkElasticsearch(ctx context.Context) (history *domain.ElasticsearchCheckHistory) {
	_uuid := uuid.New().String()
	history = new(domain.ElasticsearchCheckHistory)
	history.FillPrivateComponent()
	history.UUID = _uuid

	cluster, err := ecu.elasticsearchAgency.GetClusterHealth()
	if err != nil {
		history.ProcessLevel.Set(errorLevel)
		history.SetError(errors.Wrap(err, "failed to get cluster health"))
		msg := "!elasticsearch check error occurred! unable to get cluster health"
		history.SetAlarmResult(ecu.slackChatAgency.SendMessage("x", msg, _uuid))
		return
	}
	history.SetClusterHealth(cluster)
	var totalShards = intComparator{V: cluster.ActiveShards() + cluster.UnassignedShards()}

	switch ecu.status {
	case elasticsearchStatusHealthy:
		break
	case elasticsearchStatusRecovering:
		history.ProcessLevel.Set(recoveringLevel)
		history.Message = "recovering elasticsearch health is already on process"
		return
	case elasticsearchStatusUnhealthy:
		if totalShards.isLessThan(ecu.myCfg.MaximumShardsNumber()) {
			ecu.setStatus(elasticsearchStatusHealthy)
			history.ProcessLevel.Set(recoveredLevel)
			history.Message = "elasticsearch check is recovered to be healthy"
			msg := fmt.Sprintf("!elasticsearch check recovered to health! total shards - %d", totalShards.V)
			_, _, _ = ecu.slackChatAgency.SendMessage("heart", msg, _uuid)
		} else {
			history.ProcessLevel.Set(unhealthyLevel)
			history.Message = "elasticsearch check is unhealthy now"
		}
		return
	}

	if totalShards.isMoreThan(ecu.myCfg.MaximumShardsNumber()) {
		ecu.setStatus(elasticsearchStatusRecovering)
		history.ProcessLevel.Set(weakDetectedLevel)
		msg := "!elasticsearch check weak detected! start to delete jaeger index"
		history.SetAlarmResult(ecu.slackChatAgency.SendMessage("pill", msg, _uuid))

		indices, err := ecu.elasticsearchAgency.GetIndicesWithPatterns([]string{ecu.myCfg.JaegerIndexPattern()})
		if err != nil {
			ecu.setStatus(elasticsearchStatusUnhealthy)
			history.ProcessLevel.Append(errorLevel)
			msg := "!elasticsearch check error occurred! failed to get indices, please check for yourself"
			_, _, _ = ecu.slackChatAgency.SendMessage("broken_heart", msg, _uuid)
			history.SetError(errors.Wrap(err, "failed to get indices with pattern"))
			return
		}
		indices.SetMinLifeCycle(ecu.myCfg.JaegerIndexMinLifeCycle())

		if err := ecu.elasticsearchAgency.DeleteIndices(indices.IndexNames()); err != nil {
			ecu.setStatus(elasticsearchStatusUnhealthy)
			history.ProcessLevel.Append(errorLevel)
			msg := "!elasticsearch check error occurred! failed to delete indices, please check for yourself"
			_, _, _ = ecu.slackChatAgency.SendMessage("anger", msg, _uuid)
			history.SetError(errors.Wrap(err, "failed to delete indices"))
			return
		} else {
			history.IfJaegerIndexDeleted = true
			history.DeletedJaegerIndices = indices.IndexNames()
			history.Message = "pruned docker system as current disk capacity is less than the minimum"
		}

		againCluster, err := ecu.elasticsearchAgency.GetClusterHealth()
		if err != nil {
			ecu.setStatus(elasticsearchStatusUnhealthy)
			history.ProcessLevel.Append(errorLevel)
			msg := "!elasticsearch check error occurred! failed to again get cluster health, please check for yourself"
			_, _, _ = ecu.slackChatAgency.SendMessage("broken_heart", msg, _uuid)
			history.SetError(errors.Wrap(err, "failed to again get cluster health again"))
			return
		}
		history.SetClusterHealth(againCluster)
		var againTotalShards = intComparator{V: againCluster.ActiveShards() + againCluster.UnassignedShards()}

		if againTotalShards.isLessThan(ecu.myCfg.MaximumShardsNumber()) {
			ecu.setStatus(elasticsearchStatusHealthy)
			msg := fmt.Sprintf("!elasticsearch check is recovered! total shards - %d", againTotalShards.V)
			_, _, _ = ecu.slackChatAgency.SendMessage("heart", msg, _uuid)
		} else {
			ecu.setStatus(elasticsearchStatusUnhealthy)
			msg := "!elasticsearch check has deteriorated! please check for yourself"
			_, _, _ = ecu.slackChatAgency.SendMessage("broken_heart", msg, _uuid)
		}
	} else {
		history.ProcessLevel.Set(healthyLevel)
		history.Message = "elasticsearch service is healthy now"
	}

	return
}

// setStatus set status field value using mutex Lock & Unlock
func (ecu *elasticsearchCheckUsecase) setStatus(status elasticsearchCheckStatus) {
	ecu.mutex.Lock()
	defer ecu.mutex.Unlock()
	ecu.status = status
}
