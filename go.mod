module github.com/hashicorp/packer-plugin-azure

go 1.18

require (
	github.com/Azure/azure-sdk-for-go v66.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.29
	github.com/Azure/go-autorest/autorest/adal v0.9.23
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.4
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/approvals/go-approval-tests v0.0.0-20210131072903-38d0b0ec12b1
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/go-cmp v0.5.9
	github.com/hashicorp/go-azure-helpers v0.56.0
	github.com/hashicorp/hcl/v2 v2.16.2
	github.com/hashicorp/packer-plugin-sdk v0.4.0
	github.com/masterzen/winrm v0.0.0-20210623064412-3b76017826b0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mitchellh/reflectwalk v1.0.2
	github.com/stretchr/testify v1.8.2
	github.com/zclconf/go-cty v1.13.1
	golang.org/x/crypto v0.9.0
)

require (
	github.com/dimchansky/utfbom v1.1.1
	github.com/hashicorp/go-azure-sdk v0.20230523.1140858
	github.com/mitchellh/go-homedir v1.1.0
	github.com/tombuildsstuff/giovanni v0.20.0
)

require (
	cloud.google.com/go v0.105.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.0 // indirect
	cloud.google.com/go/iam v0.6.0 // indirect
	cloud.google.com/go/storage v1.27.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20200615164410-66371956d46c // indirect
	github.com/ChrisTrenkamp/goxpath v0.0.0-20210404020558-97928f7e12b6 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-dump v0.0.0-20190214190832-042adf3cf4a0 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/armon/go-metrics v0.3.9 // indirect
	github.com/aws/aws-sdk-go v1.44.114 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dylanmei/iso8601 v0.1.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.0 // indirect
	github.com/googleapis/gax-go/v2 v2.6.0 // indirect
	github.com/hashicorp/consul/api v1.10.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320 // indirect
	github.com/hashicorp/go-getter/gcs/v2 v2.2.0 // indirect
	github.com/hashicorp/go-getter/s3/v2 v2.2.0 // indirect
	github.com/hashicorp/go-getter/v2 v2.2.0 // indirect
	github.com/hashicorp/go-hclog v1.4.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.2 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-safetemp v1.0.0 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/serf v0.9.5 // indirect
	github.com/hashicorp/terraform-plugin-go v0.14.3 // indirect
	github.com/hashicorp/terraform-plugin-log v0.8.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.26.1 // indirect
	github.com/hashicorp/vault/api v1.1.1 // indirect
	github.com/hashicorp/vault/sdk v0.2.1 // indirect
	github.com/hashicorp/yamux v0.0.0-20210826001029-26ff87cf9493 // indirect
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.11.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/masterzen/simplexml v0.0.0-20190410153822-31eea3082786 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-fs v0.0.0-20180402235330-b7b9ca407fff // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/iochan v1.0.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/packer-community/winrmcp v0.0.0-20180921211025-c76d91c1e7db // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/sftp v1.13.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/ugorji/go/codec v1.2.6 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.12 // indirect
	github.com/vmihailenco/tagparser v0.1.1 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/oauth2 v0.4.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.101.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c // indirect
	google.golang.org/grpc v1.51.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	software.sslmate.com/src/go-pkcs12 v0.2.0 // indirect
)

replace github.com/zclconf/go-cty => github.com/nywilken/go-cty v1.10.1-0.20230602202310-ae904726bfe1

// Incorrect plugin registration for Azure component; see hashicorp/packer-plugin-azure/pull/73
retract v0.0.1
