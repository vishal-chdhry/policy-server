package inmemory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/k3s-io/kine/pkg/client"
	"github.com/kyverno/policy-server/pkg/utils"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type inMemoryDb struct {
	sync.Mutex

	logger *zap.SugaredLogger
	db     map[string]client.Value
}

func New(logger *zap.SugaredLogger) client.Client {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	inMemoryDb := &inMemoryDb{
		db:     make(map[string]client.Value),
		logger: logger.Named("inmemoryDb"),
	}
	return inMemoryDb
}

func (i *inMemoryDb) List(ctx context.Context, prefix string, rev int) ([]client.Value, error) {
	i.Lock()
	defer i.Unlock()

	i.logger.Infof("listing all values for prefix:%s", prefix)
	res := make([]client.Value, 0)

	for k, v := range i.db {
		if strings.HasPrefix(k, prefix) {
			res = append(res, v)
			i.logger.Infof("value found for prefix:%s, key:%s, valuelength:%d", prefix, k, len(v.Data))
		}
	}

	return res, nil
}

func (i *inMemoryDb) Get(ctx context.Context, key string) (client.Value, error) {
	i.Lock()
	defer i.Unlock()

	i.logger.Infof("getting value for key:%s", key)
	if val, ok := i.db[key]; ok {
		i.logger.Infof("value found for key:%s valuelength:%d", key, len(val.Data))
		return val, nil
	} else {
		i.logger.Errorf("value not found for key:%s", key)
		return client.Value{}, errors.NewNotFound(schema.GroupResource{Group: utils.GroupVersion, Resource: ""}, key)
	}
}

func (i *inMemoryDb) Put(ctx context.Context, key string, value []byte) error {
	i.Lock()
	defer i.Unlock()

	i.logger.Infof("putting data for key:%s valuelength:%d", key, len(value))
	i.db[key] = client.Value{
		Key:      []byte(key),
		Data:     value,
		Modified: time.Now().Unix(),
	}
	i.logger.Infof("value put for key:%s", key)

	return nil
}

func (i *inMemoryDb) Create(ctx context.Context, key string, value []byte) error {
	i.Lock()
	defer i.Unlock()

	i.logger.Infof("creating entry for key:%s valuelength:%d", key, len(value))
	if _, found := i.db[key]; found {
		i.logger.Errorf("entry already exists k:%s", key)
		return errors.NewAlreadyExists(schema.GroupResource{Group: utils.GroupVersion, Resource: ""}, key)
	} else {
		i.db[key] = client.Value{
			Key:      []byte(key),
			Data:     value,
			Modified: time.Now().Unix(),
		}
		i.logger.Infof("entry created for key:%s", key)
		return nil
	}
}

func (i *inMemoryDb) Update(ctx context.Context, key string, revision int64, value []byte) error {
	i.Lock()
	defer i.Unlock()

	i.logger.Infof("updating entry for key:%s valuelength:%d", key, len(value))
	if _, found := i.db[key]; !found {
		i.logger.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(schema.GroupResource{Group: utils.GroupVersion, Resource: ""}, key)
	} else {
		i.db[key] = client.Value{
			Key:      []byte(key),
			Data:     value,
			Modified: time.Now().Unix(),
		}
		i.logger.Infof("entry updated for key:%s", key)
		return nil
	}
}

func (i *inMemoryDb) Delete(ctx context.Context, key string, revision int64) error {
	i.Lock()
	defer i.Unlock()

	i.logger.Infof("deleting entry for key:%s valuelength:%d", key)
	if _, found := i.db[key]; !found {
		i.logger.Errorf("entry does not exist k:%s", key)
		return errors.NewNotFound(schema.GroupResource{Group: utils.GroupVersion, Resource: ""}, key)
	} else {
		delete(i.db, key)
		i.logger.Infof("entry deleted for key:%s", key)
		return nil
	}
}

func (i *inMemoryDb) Close() error {
	i.db = nil
	return nil
}
