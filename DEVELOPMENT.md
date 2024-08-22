# Developing `terraform-provider-datarobot`

## Installation

1. In a terminal clone the `terraform-provider-datarobot` repository.

    ~~~ shell
    git clone https://github.com/datarobot-community/terraform-provider-datarobot.git
    ~~~

1. Navigate to the `terraform-provider-datarobot` directory.

    ~~~ shell
    cd terraform-provider-datarobot
    ~~~

1. Build the binary and copy it to your path.

    ~~~ shell
    make install
    ~~~

## Environment Variables

To override the datarobot api token used by the terraform provider

~~~shell
export DATAROBOT_API_KEY=<YOUR_API_KEY>
~~~

## Regenerating Docs

Documentations is generated based on the `Description` and `MarkdownDescription` fields in the schemas.

~~~shell
make generate
~~~

# Contributor Guide

## Suggested Reading

- [Providers](https://developer.hashicorp.com/terraform/language/providers)
- [Provider Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Debugging](https://developer.hashicorp.com/terraform/plugin/debugging)
- [Testing](https://developer.hashicorp.com/terraform/plugin/testing)
- [Best practice design principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)

## Provider Philosophy

In a nutshell, a Terraform provider tells Terraform how to manage external resources and read from data sources.
Terraform users specify their desired state of the world in one or more [config files](https://developer.hashicorp.com/terraform/language/syntax/configuration).
Terraform then uses the appropriate providers to read the actual state of the world and attempts to use CRUD operations
to make the state match the plan.

## Editor Support

VSCode has a
[Terraform extension](https://marketplace.visualstudio.com/items?itemName=HashiCorp.terraform)
written by Hashicorp which I recommend.

### Asynchronous Operations

Terraform operations need to be fully blocking. We can't create a model and immediately declare the
resource as created, because that would signal to Terraform that it's ok to start creating dependent resources. Usually,
this means API polling. There are some good examples of how to do this in the custom model and deployment resources.

## Testing

We have two types of required tests for each resource and data source: Integration tests and acceptance tests. Both
typically call into a shared test method where we specify a series of configs and checks for each step, then run them
with the Terraform acceptance testing framework. The difference is that the integration tests use a mocked API client
library, while the acceptance tests are live and create real resources. The integration test wrapper needs to set
expected mock calls before calling that common method.

Integration and unit tests are run on every commit, while acceptance tests are only automatically run during the release
process. Unit tests are great, but rare since TF provider methods don't usually contain complex business logic.

**It's generally a good idea to run acceptance tests affected by your change manually before submitting a PR.** They can be run individually, setting `TF_ACC=1` and `DATAROBOT_API_KEY` to appropriate values. Please ensure the service account associated with the provided `DATAROBOT_API_KEY` has the necessary features enabled to run the tests.

Getting the mock calls right for integration tests can be challenging. You may be surprised at how many read calls are
issued at each stage. Some optional logging has been enabled which can help with this. Enable it by setting the env var `TRACE_API_CALLS=1`. If you first run an acceptance test that performs the real operations, ie `TF_ACC=1 TRACE_API_CALLS=1`, you can then use this record to determine the mock calls.

Tests should generally cover all of a resource's CRUD operations, plus `import`. That generally means at least three test steps. Data sources can either have their own simple test method or, if convenient, be dropped into an existing resource test.

There are currently no permanent fixtures for acceptance tests to use. Each test needs to create the resources it depends on.

## Questions?

If you're an external contributor, first of all, thank you! The best way to contact us is with a GitHub issue. Any and all feedback is welcome.