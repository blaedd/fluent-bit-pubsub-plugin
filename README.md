# fluent-bit output plugin for Google PubSub

This plugin allows publishing log records from fluent-bit to Google Cloud PubSub.

## Usage

`fluent-bit -e /path/to/flb_pubsub.so -c fluent-bit.conf`

## Configuration

### Example:

```
 [OUTPUT]
 match *
 name pubsub
 gcp_project_id noponies4u_logs
 topic_id fluent_logs
 credentials_file /etc/fluent-bit/gcs.json
```

### Options

#### General Options

| Option Name           | Description                                                                                                                                                     | Type                    | Default | Example                     |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------|---------|-----------------------------|
| **gcp_project_id**    | Google Cloud project id                                                                                                                                         | string                  | None    | my_gcp_project              |
| **topic_id**          | PubSub topic ID                                                                                                                                                 | string                  | None    | fluentbit_logs              |
| credentials_file      | Path to service account credentials file.                                                                                                                       | string                  | None    | /etc/fluent-bit/gcloud.json |
| timestamp_field       | Log record field to populate/update with the fluent-bit timestamp                                                                                               | string                  | None    | fb_ts                       |
| attribute_fields      | Comma seperated list of fields to use as PubSub message attributes. These are useful since subscribers can filter messages by attributes, but not body content. | comma seperated strings | None    | loghost,tag,app             |
| keep_attribute_fields | If set to true, record fields used as attributes are also left in the log record. Otherwise, they are removed.                                                  | boolean                 | false   | true                        |
| publish_timeout       | Timeout to use on the PubSub publisher client.                                                                                                                  | Duration                | 60s     | 2m                          |

**Indicates required field**

If a credentials file isn't provided, the library attempts to
use [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials)

#### Batch options

These correspond to [PublishSettings](https://pkg.go.dev/cloud.google.com/go/pubsub#PublishSettings).

| Option Name             | Description                                                  | Type     | Default   |
|-------------------------|--------------------------------------------------------------|----------|-----------|
| publish_delay_threshold | Publish a non-empty batch after this many seconds has passed | Duration | 1s        |
| publish_byte_threshold  | Publish a batch once it reaches this size in bytes.          | int      | 1,000,000 |
| publish_count_threshold | Publish a batch once it has this many messages.              | int      | 100       |

## Build

### Linux/Darwin/etc

go build -buildmode=c-shared -o flb_pubsub.so .

### Windows

* Install [MSYS2](https://www.msys2.org/) or another distribution such as Cygwin with mingw-gcc
* Install appropriate gcc package (32bit or 64bit depending on what fluent-bit package you have installed)
* Add MSYS2 bin directory with the appropriate compiler to your path (e.g. C:\MSYS2\mingw64\bin)
* go build -buildmode=c-shared -o flb_pubsub.dll .


