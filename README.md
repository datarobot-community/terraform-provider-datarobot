# Terraform provider for DataRobot (Preview)

`terraform-provider-datarobot` is the [Terraform provider](https://developer.hashicorp.com/terraform/language/providers) for the [DataRobot API](https://docs.datarobot.com/en/docs/api/index.html).

- [More information about DataRobot](https://www.datarobot.com/)
- [More information about Terraform](https://terraform.io/)

For information on developing `terraform-provider-datarobot` see [DEVELOPMENT.md](https://github.com/datarobot-community/terraform-provider-datarobot/blob/main/DEVELOPMENT.md)

## Getting Started

### Prerequisistes

Before you use `terraform-provider-datarobot` you must [install Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli) and [git](https://git-scm.com/downloads) on your local machine.


## Run the Low-Code Monitored RAG Example

1. In a terminal clone the `terraform-provider-datarobot` repository:

    ~~~ shell
    git clone https://github.com/datarobot-community/terraform-provider-datarobot.git
    ~~~

Since the provider has not been published to the Terraform Registry yet, you must also:

1. Ensure [Go](https://go.dev/doc/install) >= 1.16 is installed.

1. Run `cd terraform-provider-datarobot` and then `go mod tidy`

1. Run `make install` to build the provider locally. 

Now you can continue with the Low Code RAG example:

1. Go to the `examples/workflows/low_code_rag` directory.

    ~~~ shell
    cd examples/workflows/low_code_rag
    ~~~

1. The provider requires an API key set in an environment variable named `DATROBOT_API_TOKEN`. Copy the [API key](https://docs.datarobot.com/en/docs/get-started/acct-mgmt/acct-settings/api-key-mgmt.html#api-key-management) from the DataRobot console and create the `DATAROBOT_API_TOKEN` environment variable.

    ~~~ shell
    export DATAROBOT_API_TOKEN=<YOUR_API_KEY>
    ~~~

    Where `<your API key>` is the API key you copied from the DataRobot Console.
 
 1. The example requires Google Cloud service account credentials in order to call the Google Vertex AI API. Follow [this guide](https://cloud.google.com/iam/docs/keys-create-delete#creating) to create a service account key for your Google account.

 1. In a text editor create a new file `terraform.tfvars` in `low_code_rag` with the following settings.

     ~~~
    use_case_name = "<use case name>"
    google_cloud_credential_source_file = "<source_file>"
    ~~~

    Where:  
        - `<use case name>` is the name of the use case you want to create.
        - `<source file>` is the path to your Google Cloud service account key file.

1. Initialize the provider.

    ~~~ shell
    terraform init
    ~~~

    This reads the `main.tf` configuration file, which contains the information on how the provider will create the Datarobot resources. The `terraform.tfvars` file sets the use case name.

1. Create the Terraform plan. This shows the actions the provider will take, but won't perform them.

    ~~~ shell
    terraform plan
    ~~~

1. Create the resources.

    ~~~ shell
    terraform apply
    ~~~

    Enter `yes` when prompted to apply the plan and create the resources.

1. Once the creation is complete, navigate to the `datarobot_qa_application_url` to view your Q&A application.

    ~~~ shell
    Apply complete! Resources: 5 added, 0 changed, 0 destroyed.

    Outputs:

    datarobot_qa_application_url = "<your_qa_application_url>"
    ~~~

1. (optional) Delete the resources when you are done.

    ~~~ shell
    terraform destroy
    ~~~

    Enter `yes` when prompted to delete the resources.
