// Create file in v.1.0.0
// srvcheck_swarmpit_ucase.go is file that define usecase implementation about swarmpit check in srvcheck domain
// usecase layer depend on repository layer and is depended to delivery layer

package usecase

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/google/uuid"
	"github.com/inhies/go-bytesize"
	"github.com/pkg/errors"
	"sync"

	"github.com/DMS-SMS/v1-health-check/domain"
)

// swarmpitCheckStatus is type to int constant represent current swarmpit check process status
type swarmpitCheckStatus int
const (
	swarmpitStatusHealthy    swarmpitCheckStatus = iota // represent swarmpit check status is healthy
	swarmpitStatusRecovering                            // represent it's recovering swarmpit status now
	swarmpitStatusUnhealthy                             // represent swarmpit check status is unhealthy
)

// swarmpitCheckUsecase implement SwarmpitCheckUsecase interface in domain and used in delivery layer
type swarmpitCheckUsecase struct {
	// myCfg is used for getting swarmpit check usecase config
	myCfg swarmpitCheckUsecaseConfig

	// historyRepo is used for store swarmpit check history and injected from outside
	historyRepo domain.SwarmpitCheckHistoryRepository

	// slackChat is used for agent slack API about chatting
	slackChatAgency slackChatAgency

	// dockerAgency is used as agency about docker engine API
	dockerAgency dockerAgency

	// status represent current process status of swarmpit health check
	status swarmpitCheckStatus

	// mutex help to prevent race condition when set status field value
	mutex sync.Mutex
}

// swarmpitCheckUsecaseConfig is the config getter interface for swarmpit check usecase
type swarmpitCheckUsecaseConfig interface {
	// get common config method from embedding serviceCheckUsecaseComponentConfig
	serviceCheckUsecaseComponentConfig

	// SwarmpitAppServiceName method returns string represent swarmpit app service name
	SwarmpitAppServiceName() string

	// SwarmpitAppMaxMemoryUsage method returns bytesize represent swarmpit app maximum memory usage
	SwarmpitAppMaxMemoryUsage() bytesize.ByteSize
}

// NewSwarmpitCheckUsecase function return swarmpitCheckUsecase ptr instance after initializing
func NewSwarmpitCheckUsecase(
	cfg swarmpitCheckUsecaseConfig,
	shr domain.SwarmpitCheckHistoryRepository,
	sca slackChatAgency,
	da dockerAgency,
) domain.SwarmpitCheckUseCase {
	return &swarmpitCheckUsecase{
		// initialize field with parameter received from caller
		myCfg:           cfg,
		historyRepo:     shr,
		slackChatAgency: sca,
		dockerAgency:    da,

		// initialize field with default value
		status: swarmpitStatusHealthy,
		mutex:  sync.Mutex{},
	}
}

// CheckSwarmpit check swarmpit health with checkSwarmpit method & store check history in repository
// Implement CheckSwarmpit method of SwarmpitCheckUseCase interface
func (scu *swarmpitCheckUsecase) CheckSwarmpit(ctx context.Context) (err error) {
	history := scu.checkSwarmpit(ctx)

	if b, err := scu.historyRepo.Store(history); err != nil {
		return errors.Wrapf(err, "failed to store swarmpit check history, response: %s", string(b))
	}

	return
}

// method processed with below logic about swarmpit health check according to current check status
// 0 : 정상적으로 인지된 상태 (상태 확인 수행) (SwarmpitApp 컨테이너 메모리 사용량 기준)
// 0 -> 1 : SwarmpitApp 재시작 실행 (SwarmpitApp 재시동 알림 발행)
// 1 : SwarmpitApp 재시작증 (상태 확인 수행 X)
// 1 -> 0 : SwarmpitApp 재시작으로 인해 상태 회복 완료 (상태 회복 알림 발행)
// 1 -> 2 : SwarmpitApp 재시작을 해도 상태 회복 X (상태 회복 불가능 상태 알림 발행)
// 2 : 관리자가 직접 확인해야함 (상태 확인 수행 X)
// 2 -> 0 : 관리자 직접 상태 회복 완료 (상태 회복 알림 발행)
func (scu *swarmpitCheckUsecase) checkSwarmpit(ctx context.Context) (history *domain.SwarmpitCheckHistory) {
	_uuid := uuid.New().String()
	history = new(domain.SwarmpitCheckHistory)
	history.FillPrivateComponent()
	history.UUID = _uuid

	ctn, err := scu.dockerAgency.GetContainerWithServiceName(scu.myCfg.SwarmpitAppServiceName())
	if err != nil {
		history.ProcessLevel.Set(errorLevel)
		history.SetError(errors.Wrap(err, "failed to get swarmpit app docker container"))
		msg := "!swarmpit check error occurred! unable to get swarmpit app container"
		history.SetAlarmResult(scu.slackChatAgency.SendMessage("x", msg, _uuid))
		return
	}
	history.SwarmpitAppMemoryUsage = ctn.MemoryUsage()
	var memoryUsage = bytesizeComparator{V: ctn.MemoryUsage()}

	switch scu.status {
	case swarmpitStatusHealthy:
		break
	case swarmpitStatusRecovering:
		history.ProcessLevel.Set(recoveringLevel)
		history.Message = "recovering swarmpit health is already on process"
		return
	case swarmpitStatusUnhealthy:
		if memoryUsage.isLessThan(scu.myCfg.SwarmpitAppMaxMemoryUsage()) {
			scu.setStatus(swarmpitStatusHealthy)
			history.ProcessLevel.Set(recoveredLevel)
			history.Message = "swarmpit check is recovered to be healthy"
			msg := fmt.Sprintf("!swarmpit check recovered to health! memory usage - %s", memoryUsage.V)
			_, _, _ = scu.slackChatAgency.SendMessage("heart", msg, _uuid)
		} else {
			history.ProcessLevel.Set(unhealthyLevel)
			history.Message = "swarmpit check is unhealthy now"
		}
		return
	}

	if memoryUsage.isMoreThan(scu.myCfg.SwarmpitAppMaxMemoryUsage()) {
		scu.setStatus(swarmpitStatusRecovering)
		history.ProcessLevel.Set(weakDetectedLevel)
		msg := "!swarmpit check weak detected! start to restart swarmpit app"
		history.SetAlarmResult(scu.slackChatAgency.SendMessage("pill", msg, _uuid))

		if err := scu.dockerAgency.RemoveContainer(ctn.ID(), types.ContainerRemoveOptions{Force: true}); err != nil {
			scu.setStatus(swarmpitStatusUnhealthy)
			history.ProcessLevel.Append(errorLevel)
			msg := "!swarmpit check error occurred! failed to remove swarmpit app, please check for yourself"
			_, _, _ = scu.slackChatAgency.SendMessage("anger", msg, _uuid)
			history.SetError(errors.Wrap(err, "failed to remove swarmpit app"))
			return
		} else {
			scu.setStatus(swarmpitStatusHealthy)
			history.IfSwarmpitAppRestarted = true
			history.Message = "restart swarmpit app as swarmpit app memory usage is more than the maximum"
			msg := "!swarmpit check is recovered! succeed to restart swarmpit app"
			_, _, _ = scu.slackChatAgency.SendMessage("heart", msg, _uuid)
		}
	} else {
		history.ProcessLevel.Set(healthyLevel)
		history.Message = "swarmpit service is healthy now"
	}

	return
}

// setStatus set status field value using mutex Lock & Unlock
func (scu *swarmpitCheckUsecase) setStatus(status swarmpitCheckStatus) {
	scu.mutex.Lock()
	defer scu.mutex.Unlock()
	scu.status = status
}
