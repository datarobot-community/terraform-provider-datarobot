# Terraform provider for DataRobot (Preview)

`terraform-provider-datarobot` is a [Terraform provider](https://developer.hashicorp.com/terraform/language/providers) for interacting with the [DataRobot API](https://docs.datarobot.com/en/docs/api/index.html).

## Resources

- [Learn more about DataRobot](https://www.datarobot.com/)
- [Learn more about Terraform](https://terraform.io/)
- For development details, see [DEVELOPMENT.md](https://github.com/datarobot-community/terraform-provider-datarobot/blob/main/DEVELOPMENT.md).

## Getting Started

### Prerequisites

Before using `terraform-provider-datarobot`, ensure the following tools are installed on your local machine:

- [Terraform CLI](https://learn.hashicorp.com/tutorials/terraform/install-cli)
- [Git](https://git-scm.com/downloads)

### Running the Low-Code Monitored RAG Example

1. **Clone the repository**:
  ```bash
  git clone https://github.com/datarobot-community/terraform-provider-datarobot.git
  ```

2. **Prepare the environment**:
  - Install [Go](https://go.dev/doc/install) (version >= 1.16).
  - Navigate to the repository directory:
    ```bash
    cd terraform-provider-datarobot
    ```
  - Run:
    ```bash
    go mod tidy
    make install
    ```

3. **Set up the example**:
  - Navigate to the desired example directory:
    ```bash
    cd examples/workflows/low_code_rag
    ```
    or
    ```bash
    cd examples/workflows/notebooks
    ```

  - Set the `DATAROBOT_API_TOKEN` environment variable with your [DataRobot API key](https://docs.datarobot.com/en/docs/get-started/acct-mgmt/acct-settings/api-key-mgmt.html#api-key-management):
    ```bash
    export DATAROBOT_API_TOKEN=<YOUR_API_KEY>
    ```

  - For the RAG example, create a `terraform.tfvars` file in the `low_code_rag` directory with the following content:
    ```hcl
    use_case_name = "<use case name>"
    google_cloud_credential_source_file = "<source_file>"
    ```
    Replace `<use case name>` with your desired use case name and `<source_file>` with the path to your Google Cloud service account key file.

4. **Initialize and apply the Terraform configuration**:
  - Initialize the provider:
    ```bash
    terraform init
    ```
  - Preview the changes:
    ```bash
    terraform plan
    ```
  - Apply the configuration:
    ```bash
    terraform apply
    ```
    Confirm with `yes` when prompted.

5. **Access the resources**:
  - After successful execution, access the generated resources using the provided URLs:
    ```bash
    Apply complete! Resources: 5 added, 0 changed, 0 destroyed.

    Outputs:

    datarobot_qa_application_url = "<your_qa_application_url>"
    ```

6. **Clean up (optional)**:
  - To delete the resources:
    ```bash
    terraform destroy
    ```
    Confirm with `yes` when prompted.
