if [ -z "${AZURE_CLI_AUTH}" ]; then
    echo "AZURE_CLI_AUTH is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_RESOURCE_GROUP_NAME}" ]; then
    echo "ARM_RESOURCE_GROUP_NAME is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_STORAGE_ACCOUNT}" ]; then
    echo "ARM_STORAGE_ACCOUNT is unset or set to the empty string"
    exit 1
fi

if [ -z "${ARM_RESOURCE_PREFIX}" ]; then
    echo "ARM_RESOURCE_PREFIX is unset or set to the empty string"
    exit 1
fi

if ! command -v terraform &> /dev/null
then
    echo "terraform is not installed"
    exit 1
fi
terraform destroy -var "resource_prefix=${ARM_RESOURCE_PREFIX}" -var "resource_group_name=${ARM_RESOURCE_GROUP_NAME}" -var "storage_account_name=${ARM_STORAGE_ACCOUNT}" -auto-approve

