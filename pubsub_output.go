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

package main

import "C"
import (
	"context"
	"io"
	"os"
	"strconv"
	"time"
	"unsafe"

	"cloud.google.com/go/pubsub"
	"github.com/blaedd/fluent-bit-pubsub-plugin/plugin"
	"github.com/fluent/fluent-bit-go/output"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	PluginName = "pubsub"
)

var (
	pluginInstances []*plugin.OutputPlugin
)

func init() {
	zerolog.DisableSampling(true)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
	pi := zerolog.Dict()
	pi.Str("name", PluginName).Str("host", hostname)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().
		Caller().Dict("plugin", pi).Logger()
}

func addPluginInstance(ctx unsafe.Pointer) error {
	logger := log.With().Uint("plugin_ctx", uint(uintptr(ctx))).Logger().Level(zerolog.InfoLevel)
	id := len(pluginInstances)
	output.FLBPluginSetContext(ctx, id)
	cs := plugin.NewFLBConfigStore(ctx, &logger)
	cfg := plugin.BuildPluginConfig(id, &cs)
	if cfg.D {
		logger = logger.Level(zerolog.DebugLevel)
		logger.Debug().Msg("debug logging enabled")
	}
	reqCtx := context.Background()
	reqCtx = logger.WithContext(reqCtx)
	instance, err := plugin.NewPluginFromConfig(reqCtx, cfg)
	if err != nil {
		return err
	}
	pluginInstances = append(pluginInstances, instance)
	return nil
}

func getPluginInstance(ctx unsafe.Pointer) *plugin.OutputPlugin {
	pluginID := output.FLBPluginGetContext(ctx).(int)
	return pluginInstances[pluginID]
}

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	p := output.FLBPluginRegister(ctx, PluginName, "GCP PubSub Fluent Bit Plugin!")
	log.Info().Msg("output Plugin Registered")
	return p
}

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	log.Info().Msg("initializing output plugin.")
	err := addPluginInstance(ctx)
	if err != nil {
		log.Error().Err(err).Msg("unable to initialize plugin")
		return output.FLB_ERROR
	}
	log.Info().Msg("plugin initialized")
	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	reqCtx := context.Background()
	p := getPluginInstance(ctx)
	fluentTag := C.GoString(tag)
	logger := log.With().Str("tag", fluentTag).Logger()
	if p.D {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}
	reqCtx = logger.WithContext(reqCtx)
	logger.Debug().Int("bytes", int(length)).Msg("receiving log entries")
	p.R.ResetReader(data, int(length))
	rb := make([]*pubsub.PublishResult, 0, 100)

	for {
		ts, record, err := p.R.ReadRecord()
		if err == io.EOF {
			logger.Info().Msg("End of File")
			break
		}
		if err != nil {
			logger.Error().Err(err).Time("log_ts", ts).Msg("error while reading a record")
			continue
		}
		msg, err := p.CreateMessage(ts, fluentTag, record)
		if err != nil {
			logger.Error().Err(err).Time("log_ts", ts).Interface("record", record).Msg(
				"error while creating pubsub.Message from record")
		}
		rb = append(rb, p.Publish(reqCtx, msg))
	}

	for _, res := range rb {
		_, err := res.Get(reqCtx)
		if err != nil {
			if err == context.DeadlineExceeded {
				logger.Warn().Msg("deadline exceeded. Will retry.")
				return output.FLB_RETRY
			}
			if stts, ok := status.FromError(err); !ok {
				logger.Error().Err(err).Msg("could not parse error to grpc.status.")
				return output.FLB_ERROR
			} else {
				switch stts.Code() {
				case codes.DeadlineExceeded, codes.Internal, codes.Unavailable:
					logger.Warn().Err(err).Msg("retryable error.")
					return output.FLB_RETRY
				default:
					logger.Warn().Err(stts.Err()).Msg("unrecoverable Publish error")
					return output.FLB_ERROR
				}
			}
		}
	}
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	log.Info().Msg("exiting")
	return output.FLB_OK
}

func main() {}
