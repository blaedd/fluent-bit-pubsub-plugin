/*
 * Copyright 2022  David MacKinnon (blaedd@gmail.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"). You may
 * not use this file except in compliance with the License. A copy of the
 * License is located at
 *
 * https://www.apache.org/licenses/LICENSE-2.0.txt
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package plugin

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/api/option"
)

// ConfigStore defines an interface to the plugin configuration.
type ConfigStore interface {
	Bool(name string) (bool, bool)
	Duration(name string) (time.Duration, bool)
	Int(name string) (int, bool)
	String(name string) (string, bool)
	Strings(name string) ([]string, bool)
}

// OutputPluginConfig represents the configuration used to create an [OutputPlugin]
type OutputPluginConfig struct {
	ID      int                    // Plugin ID.
	PID     string                 // Google Cloud project id.
	TID     string                 // PubSub topic ID.
	Crds    string                 // Google Cloud credentials file.
	TSField string                 // Field to populate/update with fluent-bit timestamp.
	As      []string               // List of record fields to use as PubSub.Message attributes
	KA      bool                   // If record fields used as attributes should be kept in the record.
	PS      pubsub.PublishSettings // Pubsub PublishSettings
	D       bool                   // Debug flag
}

// Validate validates that all required fields are present in the OutputPluginConfig.
func (c *OutputPluginConfig) Validate() error {
	errorStr := "%s is a required parameter"
	var err error

	if c.PID == "" {
		err = fmt.Errorf(errorStr, "gcp_project_id")
	} else if c.TID == "" {
		err = fmt.Errorf(errorStr, "topic_id")
	}
	return err
}

func (c *OutputPluginConfig) fetchTopic(ctx context.Context, l *zerolog.Logger, client *pubsub.Client) (*pubsub.Topic, error) {
	l.Info().Msg("retrieving topic")

	topic := client.Topic(c.TID)
	topic.PublishSettings = c.PS
	ok, err := topic.Exists(ctx)
	if err != nil {
		l.Error().Err(err).Msg("unable to retrieve topic information")
		return nil, err
	}
	if ok == false {
		l.Error().Msg("topic does not exist in project")
		return nil, errors.Errorf("topic %s does not exist in project %s", c.TID, c.PID)
	}
	return topic, nil
}

func (c *OutputPluginConfig) createClient(ctx context.Context, l *zerolog.Logger, opts ...option.ClientOption) (*pubsub.Client, error) {
	if c.Crds != "" {
		l.Info().Msg("using credentials file to authenticate")
		opts = append(opts, option.WithCredentialsFile(c.Crds))
	} else {
		l.Info().Msg("no credentials file supplied. attempting to use default credentials")
	}
	client, err := pubsub.NewClient(ctx, c.PID, opts...)
	if err != nil {
		l.Error().Err(err).Msg("unable to create PubSub.Client")
		return nil, err
	}
	return client, nil
}

// BuildPluginConfig creates the OutputPluginConfig from a ConfigStore
func BuildPluginConfig(id int, cs ConfigStore) *OutputPluginConfig {
	cfg := &OutputPluginConfig{ID: id, PS: pubsub.DefaultPublishSettings}
	cfg.PS.DelayThreshold = 1 * time.Second
	cfg.D, _ = cs.Bool("debug")
	cfg.PID, _ = cs.String("gcp_project_id")
	cfg.TID, _ = cs.String("topic_id")
	cfg.Crds, _ = cs.String("credentials_file")
	cfg.TSField, _ = cs.String("timestamp_field")
	cfg.As, _ = cs.Strings("attribute_fields")
	cfg.KA, _ = cs.Bool("keep_attribute_fields")
	if val, ok := cs.Duration("publish_delay_threshold"); ok {
		cfg.PS.DelayThreshold = val
	}
	if val, ok := cs.Duration("publish_timeout"); ok {
		cfg.PS.Timeout = val
	}
	if val, ok := cs.Int("publish_byte_threshold"); ok {
		cfg.PS.ByteThreshold = val
	}
	if val, ok := cs.Int("publish_count_threshold"); ok {
		cfg.PS.CountThreshold = val
	}
	return cfg
}
