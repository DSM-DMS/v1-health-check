// Create file in v.1.0.0
// syscheck_memory_repo.go is file that define repository implement about memory using elasticsearch
// memory repository struct embed esRepositoryRequiredComponent struct in ./syscheck.go file

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

// esMemoryCheckHistoryRepository is to handle MemoryCheckHistory model using elasticsearch as data store
type esMemoryCheckHistoryRepository struct {
	// esMigrator is used for migrate elasticsearch repository in Migrate method
	esMigrator esRepositoryMigrator

	// myCfg is used for get memory check history repository config about elasticsearch
	myCfg esMemoryCheckHistoryRepoConfig

	// esCli is elasticsearch client connection injected from the outside package
	esCli *elasticsearch.Client

	// bodyWriter is implementation of reqBodyWriter interface to write []byte for request body
	bodyWriter reqBodyWriter
}

// esMemoryCheckHistoryRepoConfig is the config for memory check history repository using elasticsearch
type esMemoryCheckHistoryRepoConfig interface {
	// get common method from embedding esRepositoryComponentConfig
	esRepositoryComponentConfig
}

// NewESMemoryCheckHistoryRepository return new object that implement MemoryCheckHistoryRepository interface
func NewESMemoryCheckHistoryRepository(cfg esMemoryCheckHistoryRepoConfig, cli *elasticsearch.Client, w reqBodyWriter) domain.MemoryCheckHistoryRepository {
	repo := &esMemoryCheckHistoryRepository{
		myCfg:      cfg,
		esCli:      cli,
		bodyWriter: w,
	}

	if err := repo.Migrate(); err != nil {
		log.Fatal(errors.Wrap(err, "could not migrate repository").Error())
	}

	return repo
}

// Implement Migrate method of MemoryCheckHistoryRepository interface
func (emr *esMemoryCheckHistoryRepository) Migrate() error {
	return emr.esMigrator.Migrate(emr.myCfg, emr.esCli, emr.bodyWriter)
}

// Implement Store method of MemoryCheckHistoryRepository interface
func (emr *esMemoryCheckHistoryRepository) Store(history *domain.MemoryCheckHistory) (b []byte, err error) {
	body, _ := json.Marshal(history.DottedMapWithPrefix(""))
	if _, err = emr.bodyWriter.Write(body); err != nil {
		err = errors.Wrap(err, "failed to write map to body writer")
		return
	}

	buf := &bytes.Buffer{}
	if _, err = emr.bodyWriter.WriteTo(buf); err != nil {
		err = errors.Wrap(err, "failed to body writer WriteTo method")
		return
	}

	resp, err := (esapi.IndexRequest{
		Index:        emr.myCfg.IndexName(),
		Body:         bytes.NewReader(buf.Bytes()),
		Timeout:      time.Second * 5,
	}).Do(context.Background(), emr.esCli)

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
