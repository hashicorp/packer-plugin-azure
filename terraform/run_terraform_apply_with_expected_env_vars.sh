# Copyright IBM Corp. 2013, 2025
# SPDX-License-Identifier: MPL-2.0

if [ -z "${AZURE_CLI_AUTH}" ]; then
    echo "AZURE_CLI_AUTH is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_RESOURCE_GROUP_NAME}" ] && [ -z "${ARM_RESOURCE_GROUP_NAME_BASE}" ]; then
    echo "ARM_RESOURCE_GROUP_NAME or ARM_RESOURCE_GROUP_NAME_BASE is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_STORAGE_ACCOUNT}" ] && [ -z "${ARM_STORAGE_ACCOUNT_BASE}" ]; then
    echo "ARM_STORAGE_ACCOUNT or ARM_STORAGE_ACCOUNT_BASE is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_RESOURCE_PREFIX}" ] && [ -z "${ARM_RESOURCE_PREFIX_BASE}" ]; then
    echo "ARM_RESOURCE_PREFIX or ARM_RESOURCE_PREFIX_BASE is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_TENANT_ID}" ]; then
    echo "ARM_TENANT_ID is unset or set to the empty string"
    exit 1
fi

if [ -z "${AZURE_OBJECT_ID}" ]; then
    echo "AZURE_OBJECT_ID is unset or set to the empty string"
    exit 1
fi

if ! command -v terraform &> /dev/null
then
    echo "terraform is not installed"
    exit 1
fi
RESOURCE_GROUP_NAME_INPUT="${ARM_RESOURCE_GROUP_NAME_BASE:-${ARM_RESOURCE_GROUP_NAME}}"
STORAGE_ACCOUNT_INPUT="${ARM_STORAGE_ACCOUNT_BASE:-${ARM_STORAGE_ACCOUNT}}"
RESOURCE_PREFIX_INPUT="${ARM_RESOURCE_PREFIX_BASE:-${ARM_RESOURCE_PREFIX}}"
RESOURCE_SUFFIX_INPUT="${ARM_RESOURCE_SUFFIX:-}"

terraform apply -var "resource_prefix=${RESOURCE_PREFIX_INPUT}" -var "resource_group_name=${RESOURCE_GROUP_NAME_INPUT}" -var "storage_account_name=${STORAGE_ACCOUNT_INPUT}" -var "tenant_id=${ARM_TENANT_ID}" -var "object_id=${AZURE_OBJECT_ID}" -var "resource_suffix=${RESOURCE_SUFFIX_INPUT}" -auto-approve

