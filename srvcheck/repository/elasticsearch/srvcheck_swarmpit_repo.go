// Create file in v.1.0.0
// srvcheck_swarmpit_repo.go is file that define implement swarmpit history repository using elasticsearch
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
	"net/http"
	"time"

	"github.com/DMS-SMS/v1-health-check/domain"
)

// esSwarmpitCheckHistoryRepository is to handle SwarmpitCheckHistoryRepository model using elasticsearch as data store
type esSwarmpitCheckHistoryRepository struct {
	// myCfg is used for get swarmpit history repository config about elasticsearch
	myCfg esSwarmpitCheckHistoryRepoConfig

	// esCli is elasticsearch client connection injected from the outside package
	esCli *elasticsearch.Client

	// reqBodyWriter is implementation of reqBodyWriter interface to write []byte for request body
	reqBodyWriter reqBodyWriter
}

// esSwarmpitCheckHistoryRepoConfig is the config for swarmpit check history repository using elasticsearch
type esSwarmpitCheckHistoryRepoConfig interface {
	// get common method from embedding esRepositoryComponentConfig
	esRepositoryComponentConfig
}

// NewESSwarmpitCheckHistoryRepository return new object that implement SwarmpitCheckHistoryRepository interface
func NewESSwarmpitCheckHistoryRepository(
	cfg esSwarmpitCheckHistoryRepoConfig,
	cli *elasticsearch.Client,
	w reqBodyWriter,
) domain.SwarmpitCheckHistoryRepository {
	repo := &esSwarmpitCheckHistoryRepository{
		myCfg:         cfg,
		esCli:         cli,
		reqBodyWriter: w,
	}

	if err := repo.Migrate(); err != nil {
		log.Fatal(errors.Wrap(err, "could not migrate repository").Error())
	}

	return repo
}

// Implement Migrate method of SwarmpitCheckHistoryRepository interface
func (esr *esSwarmpitCheckHistoryRepository) Migrate() error {
	resp, err := (esapi.IndicesExistsRequest{
		Index: []string{esr.myCfg.IndexName()},
	}).Do(context.Background(), esr.esCli)

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to call IndicesExists, resp: %+v", resp))
	}

	if resp.StatusCode == http.StatusNotFound {
		if err := esr.createIndex(); err != nil {
			return errors.Wrap(err, "failed to create index")
		}
	}

	return nil
}

// createIndex method create index with name, shard number in esRepositoryComponentConfig
func (esr *esSwarmpitCheckHistoryRepository) createIndex() error {
	body := map[string]interface{}{}
	body["settings.number_of_shards"] = esr.myCfg.IndexShardNum()
	body["settings.number_of_replicas"] = esr.myCfg.IndexReplicaNum()

	b, _ := json.Marshal(body)
	if _, err := esr.reqBodyWriter.Write(b); err != nil {
		return errors.Wrap(err, "failed to write map to body writer")
	}

	buf := &bytes.Buffer{}
	if _, err := esr.reqBodyWriter.WriteTo(buf); err != nil {
		return errors.Wrap(err, "failed to body writer WriteTo method")
	}

	resp, err := (esapi.IndicesCreateRequest{
		Index:         esr.myCfg.IndexName(),
		Body:          bytes.NewReader(buf.Bytes()),
		MasterTimeout: time.Second * 5,
		Timeout:       time.Second * 5,
	}).Do(context.Background(), esr.esCli)

	return errors.Wrap(err, fmt.Sprintf("failed to call IndicesCreate, resp: %+v", resp))
}

// Implement Store method of SwarmpitCheckHistoryRepository interface
func (esr *esSwarmpitCheckHistoryRepository) Store(history *domain.SwarmpitCheckHistory) (b []byte, err error) {
	body, _ := json.Marshal(history.DottedMapWithPrefix(""))
	if _, err = esr.reqBodyWriter.Write(body); err != nil {
		err = errors.Wrap(err, "failed to write map to body writer")
		return
	}

	buf := &bytes.Buffer{}
	if _, err = esr.reqBodyWriter.WriteTo(buf); err != nil {
		err = errors.Wrap(err, "failed to body writer WriteTo method")
		return
	}

	resp, err := (esapi.IndexRequest{
		Index:        esr.myCfg.IndexName(),
		Body:         bytes.NewReader(buf.Bytes()),
		Timeout:      time.Second * 5,
	}).Do(context.Background(), esr.esCli)

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
