package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/runopsio/hoop/common/log"

	"olympos.io/encoding/edn"
)

const (
	defaultAddress = "http://localhost:3000"
)

type (
	Storage struct {
		client  http.Client
		address string
	}
	// TxEdnStruct must be a struct containing edn fields.
	// See https://github.com/go-edn/edn.
	TxEdnStruct any
	TxResponse  struct {
		TxID   int64     `edn:"xtdb.api/tx-id"`
		TxTime time.Time `edn:"xtdb.api/tx-time"`
	}
)

func New() *Storage {
	s := &Storage{client: http.Client{}, address: os.Getenv("XTDB_ADDRESS")}
	if s.address == "" {
		s.address = defaultAddress
	}
	return s
}

func (s *Storage) Address() string       { return s.address }
func (s *Storage) SetURL(xtdbURL string) { s.address = xtdbURL }

// buildTrxPutEdn build transaction put operation as string
func (s *Storage) buildTrxPutEdn(trxs ...TxEdnStruct) (string, error) {
	var trxVector []string
	for _, tx := range trxs {
		txEdn, err := edn.Marshal(tx)
		if err != nil {
			return "", err
		}
		trxVector = append(trxVector, fmt.Sprintf(`[:xtdb.api/put %v]`, string(txEdn)))
	}
	return fmt.Sprintf(`{:tx-ops [%v]}`, strings.Join(trxVector, "")), nil
}

// buildTrxEvictEdn build transaction evict operation as string
func (s *Storage) buildTrxEvictEdn(xtIDs ...string) (string, error) {
	var trxVector []string
	for _, xtID := range xtIDs {
		trxVector = append(trxVector, fmt.Sprintf(`[:xtdb.api/evict %q]`, xtID))
	}
	return fmt.Sprintf(`{:tx-ops [%v]}`, strings.Join(trxVector, "")), nil
}

// SubmitPutTx sends put transactions to the xtdb API
// https://docs.xtdb.com/clients/1.22.0/http/#submit-tx
func (s *Storage) SubmitPutTx(trxs ...TxEdnStruct) (*TxResponse, error) {
	url := fmt.Sprintf("%s/_xtdb/submit-tx", s.address)
	txOpsEdn, err := s.buildTrxPutEdn(trxs...)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(txOpsEdn))
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", "application/edn")
	req.Header.Set("accept", "application/edn")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("http response is empty")
	}
	defer resp.Body.Close()

	var txResponse TxResponse
	if resp.StatusCode == http.StatusAccepted {
		if err := edn.NewDecoder(resp.Body).Decode(&txResponse); err != nil {
			log.Warnf("error decoding transaction response, err=%v", err)
		}
		// make a best-effort to wait the transaction to sync
		if txResponse.TxID > 0 {
			if err := s.AwaitTx(txResponse.TxID); err != nil {
				log.Warnf(err.Error())
			}
		}
		return &txResponse, nil
	} else {
		data, _ := io.ReadAll(resp.Body)
		log.Printf("unknown status code=%v, body=%v", resp.StatusCode, string(data))
	}
	return nil, fmt.Errorf("received unknown status code=%v", resp.StatusCode)
}

// SubmitEvictTx sends evict transactions to the xtdb API
// https://docs.xtdb.com/clients/1.22.0/http/#submit-tx
func (s *Storage) SubmitEvictTx(xtIDs ...string) (*TxResponse, error) {
	if len(xtIDs) == 0 {
		return nil, fmt.Errorf("need at least one xt/id to evict")
	}
	url := fmt.Sprintf("%s/_xtdb/submit-tx", s.address)
	txOpsEdn, err := s.buildTrxEvictEdn(xtIDs...)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(txOpsEdn))
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", "application/edn")
	req.Header.Set("accept", "application/edn")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("http response is empty")
	}
	defer resp.Body.Close()

	var txResponse TxResponse
	if resp.StatusCode == http.StatusAccepted {
		if err := edn.NewDecoder(resp.Body).Decode(&txResponse); err != nil {
			log.Infof("error decoding transaction response, err=%v", err)
		}
		// make a best-effort to wait the transaction to sync
		if txResponse.TxID > 0 {
			if err := s.AwaitTx(txResponse.TxID); err != nil {
				log.Warnf(err.Error())
			}
		}
		return &txResponse, nil
	} else {
		data, _ := io.ReadAll(resp.Body)
		log.Infof("unknown status code=%v, body=%v", resp.StatusCode, string(data))
	}
	return nil, fmt.Errorf("received unknown status code=%v", resp.StatusCode)
}

func (s *Storage) PersistEntities(payloads []map[string]any) (int64, error) {
	url := fmt.Sprintf("%s/_xtdb/submit-tx", s.address)

	bytePayload, err := buildPersistPayload(payloads)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bytePayload))
	if err != nil {
		return 0, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		var txResponse TxResponse
		if err := json.NewDecoder(resp.Body).Decode(&txResponse); err != nil {
			log.Warnf("error decoding transaction response, err=%v", err)
		}
		// make a best-effort to wait the transaction to sync
		if txResponse.TxID > 0 {
			if err := s.AwaitTx(txResponse.TxID); err != nil {
				log.Warnf(err.Error())
			}
		}
		return txResponse.TxID, nil
	}

	return 0, errors.New("not 202")
}

func (s *Storage) GetEntity(xtId string) ([]byte, error) {
	url := fmt.Sprintf("%s/_xtdb/entity", s.address)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/edn")

	q := req.URL.Query()
	q.Add("eid", xtId)
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(resp.Body)
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown http response returned fetching entity, status=%v", resp.StatusCode)
	}
}

// AwaitTx Waits until the node has indexed a transaction that is at or past the supplied tx-id.
// Returns the most recent tx indexed by the node.
func (s *Storage) AwaitTx(txID int64) error {
	url := fmt.Sprintf("%s/_xtdb/await-tx?tx-id=%v&timeout=5000", s.address, txID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed awaiting transaction %v, err=%v", txID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed awaiting transaction %v, code=%v, response=%v",
			txID, resp.StatusCode, string(data))
	}
	return nil
}

func (s *Storage) GetEntityHistory(eid, sortOrder string, withDocs bool) ([]byte, error) {
	url := fmt.Sprintf("%s/_xtdb/entity", s.address)
	if sortOrder != "asc" && sortOrder != "desc" {
		return nil, fmt.Errorf("wrong sort order input, accept 'asc' or 'desc'")
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/edn")
	q := req.URL.Query()
	q.Add("eid", eid)
	q.Add("sort-order", sortOrder)
	q.Add("history", "true")
	q.Add("with-docs", fmt.Sprintf("%v", withDocs))
	req.URL.RawQuery = q.Encode()
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	respErr, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("unknown status code (%v), response=%v", resp.StatusCode, string(respErr))
}

func (s *Storage) QueryRaw(ednQuery []byte) ([]byte, error) {
	return s.queryRequest(ednQuery, "application/edn")
}

func (s *Storage) QueryRawAsJson(ednQuery []byte) ([]byte, error) {
	return s.queryRequest(ednQuery, "application/json")
}

func (s *Storage) Query(ednQuery []byte) ([]byte, error) {
	b, err := s.queryRequest(ednQuery, "application/edn")
	if err != nil {
		return nil, err
	}

	var p [][]map[edn.Keyword]any
	if err = edn.Unmarshal(b, &p); err != nil {
		return nil, err
	}

	r := make([]map[edn.Keyword]any, 0)
	for _, l := range p {
		r = append(r, l[0])
	}

	response, err := edn.Marshal(r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *Storage) QueryAsJson(ednQuery []byte) ([]byte, error) {
	b, err := s.queryRequest(ednQuery, "application/json")
	if err != nil {
		return nil, err
	}

	var p [][]map[string]any
	if err = json.Unmarshal(b, &p); err != nil {
		return nil, err
	}

	r := make([]map[string]any, 0)
	for _, l := range p {
		r = append(r, l[0])
	}

	response, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Sync will wait for xtdb to sync all documents
// if it reaches the timeout, it will return a 5xx error
func (s *Storage) Sync(timeout time.Duration) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	var response []string
	url := fmt.Sprintf("%s/_xtdb/sync?timeout=%v", s.address, timeout.Milliseconds())
	go func() {
	exit:
		for i := 1; ; i++ {
			select {
			case <-ctx.Done():
				response = append(response, "timeout reached")
			default:
				log.Debugf("attempt=%v - trying sync to xtdb at %v", i, url)
				resp, err := http.Get(url)
				if err != nil {
					response = append(response, fmt.Sprintf("attempt=%v, failed sync xtdb, error=%v", i, err))
					time.Sleep(time.Second * 2)
					continue
				}
				if resp.StatusCode != 200 {
					data, _ := io.ReadAll(resp.Body)
					if resp.Body != nil {
						_ = resp.Body.Close()
					}
					response = append(response, fmt.Sprintf("attempt=%v, failed sync xtdb, status=%v, response=%v",
						i, resp.StatusCode, string(data)))
					time.Sleep(time.Second * 2)
					continue
				}
				response = nil
				cancelFn()
				break exit
			}
		}
	}()
	<-ctx.Done()
	if len(response) > 0 {
		return fmt.Errorf(strings.Join(response, "; "))
	}
	return nil
}

func EntityToMap(obj any) map[string]any {
	payload := make(map[string]any)

	v := reflect.ValueOf(obj).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		xtdbName := v.Type().Field(i).Tag.Get("edn")
		if xtdbName != "" && xtdbName != "-" {
			payload[xtdbName] = f.Interface()
		}
	}
	return payload
}

func (s *Storage) queryRequest(ednQuery []byte, contentType string) ([]byte, error) {
	url := fmt.Sprintf("%s/_xtdb/query", s.address)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(ednQuery))
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", contentType)
	req.Header.Set("content-type", "application/edn")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func buildPersistPayload(payloads []map[string]any) ([]byte, error) {
	txOps := make([]any, 0)
	for _, payload := range payloads {
		txOps = append(txOps, []any{
			"put", payload,
		})
	}
	b, err := json.Marshal(map[string]any{"tx-ops": txOps})
	if err != nil {
		return nil, err
	}
	return b, nil
}
