// Create file in v.1.0.0
// srvcheck_consul_repo.go is file that define implement consul history repository using elasticsearch
// this elasticsearch repository struct embed esRepositoryRequiredComponent struct in ./srvcheck.go file

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"
	"log"
	"time"

	"github.com/DMS-SMS/v1-health-check/domain"
)

// esConsulCheckHistoryRepository is to handle ConsulCheckHistoryRepository model using elasticsearch as data store
type esConsulCheckHistoryRepository struct {
	// esMigrator is used for migrate elasticsearch repository in Migrate method
	esMigrator esRepositoryMigrator

	// myCfg is used for get consul history repository config about elasticsearch
	myCfg esConsulCheckHistoryRepoConfig

	// esCli is elasticsearch client connection injected from the outside package
	esCli *elasticsearch.Client

	// reqBodyWriter is implementation of reqBodyWriter interface to write []byte for request body
	reqBodyWriter reqBodyWriter
}

// esConsulCheckHistoryRepoConfig is the config for consul check history repository using elasticsearch
type esConsulCheckHistoryRepoConfig interface {
	// get common method from embedding esRepositoryComponentConfig
	esRepositoryComponentConfig
}

// NewESConsulCheckHistoryRepository return new object that implement ConsulCheckHistoryRepository interface
func NewESConsulCheckHistoryRepository(
	cfg esConsulCheckHistoryRepoConfig,
	cli *elasticsearch.Client,
	w reqBodyWriter,
) domain.ConsulCheckHistoryRepository {
	repo := &esConsulCheckHistoryRepository{
		myCfg:         cfg,
		esCli:         cli,
		reqBodyWriter: w,
	}

	if err := repo.Migrate(); err != nil {
		log.Fatal(errors.Wrap(err, "could not migrate repository").Error())
	}

	return repo
}

// Implement Migrate method of ConsulCheckHistoryRepository interface
func (ecr *esConsulCheckHistoryRepository) Migrate() error {
	return ecr.esMigrator.Migrate(ecr.myCfg, ecr.esCli, ecr.reqBodyWriter)
}

// Implement Store method of ConsulCheckHistoryRepository interface
func (ecr *esConsulCheckHistoryRepository) Store(history *domain.ConsulCheckHistory) (b []byte, err error) {
	body, _ := json.Marshal(history.DottedMapWithPrefix(""))
	if _, err = ecr.reqBodyWriter.Write(body); err != nil {
		err = errors.Wrap(err, "failed to write map to body writer")
		return
	}

	buf := &bytes.Buffer{}
	if _, err = ecr.reqBodyWriter.WriteTo(buf); err != nil {
		err = errors.Wrap(err, "failed to body writer WriteTo method")
		return
	}

	resp, err := (esapi.IndexRequest{
		Index:        ecr.myCfg.IndexName(),
		Body:         bytes.NewReader(buf.Bytes()),
		Timeout:      time.Second * 5,
	}).Do(context.Background(), ecr.esCli)

	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("failed to call IndexRequest, resp: %+v", resp))
		return
	} else if resp.IsError() {
		err = errors.Errorf("IndexRequest return error code, resp: %+v", resp)
		return
	}

	result := map[string]interface{}{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	b, _ = json.Marshal(result)
	return
}
