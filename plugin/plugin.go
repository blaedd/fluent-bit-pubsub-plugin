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

// Package plugin implements a fluent-bit output plugin that sends logs to Google Cloud PubSub.

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

// OutputPlugin is a fluent-bit output plugin for Google Cloud PubSub.
type OutputPlugin struct {
	// Unique ID of the plugin instance
	ID int
	// Field to create/update in the record with the fluent-bit timestamp
	TSField string
	// A list of record fields to set as [cloud.google.com/go/pubsub.Message] attributes
	As []string
	// If fields from As should be kept in the record, as well as made attributes.
	KA bool
	// Debug flag
	D bool
	// FluentBit record reader
	R *FLBRecordReader
	// PubSub Topic
	*pubsub.Topic
}

// NewPluginFromConfig creates a new [OutputPlugin] from an [OutputPluginConfig].
//
// Optionally taking some additional options for the RPC client.
func NewPluginFromConfig(ctx context.Context, config *OutputPluginConfig, opts ...option.ClientOption) (*OutputPlugin, error) {
	gcp := zerolog.Dict().Str("project_id", config.PID).Str("topic_id", config.TID).
		Str("credentials", config.Crds)
	l := log.Ctx(ctx).With().Dict("gcp", gcp).Logger()
	client, err := config.createClient(ctx, &l, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create pubsub.Client: %w", err)
	}
	topic, err := config.fetchTopic(ctx, &l, client)
	if err != nil {
		l.Error().Err(err)
		return nil, fmt.Errorf("unable to access pubsub.Topic: %w", err)
	}
	reader, err := NewFLBRecordReader()
	if err != nil {
		l.Error().Err(err)
		return nil, fmt.Errorf("unable to create record reader: %w", err)
	}
	return &OutputPlugin{
		ID: config.ID, TSField: config.TSField, As: config.As, D: config.D, KA: config.KA, R: reader,
		Topic: topic}, nil
}

// CreateMessage creates a pubsub.Message from the timestamp, tag, and record from fluent-bit.
func (p *OutputPlugin) CreateMessage(ts time.Time, tag string, record map[string]interface{}) (*pubsub.Message, error) {
	if p.TSField != "" {
		record[p.TSField] = ts.UnixMicro()
	}
	attrKeys := make([]string, 0, len(p.As))
	attrs := map[string]string{"tag": tag}
	for _, attrKey := range p.As {
		if attrVal, ok := record[attrKey]; ok {
			attrs[attrKey] = fmt.Sprint(attrVal)
			attrKeys = append(attrKeys, attrKey)
			if !p.KA {
				delete(record, attrKey)
			}
		}
	}
	j, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return &pubsub.Message{Attributes: attrs, Data: j}, nil
}
