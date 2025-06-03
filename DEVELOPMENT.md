# Developing `terraform-provider-datarobot`

### Developer Setup

To contribute to the provider, ensure the following dependencies are installed:

- [Go](https://go.dev/doc/install) >= 1.16
- [Terraform CLI](https://learn.hashicorp.com/tutorials/terraform/install-cli) >= 1.0
- [Make](https://www.gnu.org/software/make/) >= 4.0
- [Git](https://git-scm.com/downloads) >= 2.0
- [Docker](https://docs.docker.com/get-docker/) >= 20.10

#### Required GPG Setup for Signing Commits
To ensure the integrity of the codebase and to verify the authorship of commits, we require all contributors to sign their commits and tags with GPG. This is a security measure to prevent unauthorized changes to the codebase.
Configure GPG for signing commits and tags. This is required for the release process and PR reviews.
1. How to set up GPG:
   - [GPG Setup](https://docs.github.com/en/authentication/managing-commit-signature-verification/setting-up-gpg-signature-verification)
   - [GPG Key Generation](https://www.gnupg.org/documentation/manuals/gnupg/Generating-a-Keypair.html)
2. Verify your GPG key is set up correctly:
   ```bash
   gpg --list-secret-keys --keyid-format LONG
   ```
3. Copy the GPG key ID (the long string after `sec`) and add it to your Git configuration:
   ```bash
   git config --global user.signingkey <YOUR_GPG_KEY_ID>
   ```
4. Enable commit signing:
   ```bash
   git config --global commit.gpgSign true
   ```
5. Verify  your GPG setup by signing a test message:
   ```bash
   echo "test" | gpg --clearsign
   ```
   you should see a message indicating that the signature is valid, like :
   ```
   -----BEGIN PGP SIGNED MESSAGE-----
    Hash: SHA512
    test
    -----BEGIN PGP SIGNATURE-----

>
> ### !!! Nota bene
> If you already committed and pushed a commit without signing, you can amend the commit with:
> ```
> git commit --amend --no-edit -S
> ```
> or rebase the commit with:
> ```
> git rebase -i <commit_hash_where_you_started>
> ```
> and change the commit message to `edit` or `reword`, then push force it with:
> ```bash
> git push --force-with-lease
> ```


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

Define the required environment variables in your terminal or via a `.env` file in the root directory.

~~~shell
export DATAROBOT_API_TOKEN=<YOUR_API_KEY>
~~~


## Updating Documentation and Examples

### Update CHANGELOG.md

1. Add a new entry in `CHANGELOG.md` to document the changes for the new version:
  ```markdown
  ## [vX.Y.Z] - YYYY-MM-DD
  ### Added
  - Added new example for Low-Code Monitored RAG in `/examples/workflows/low_code_rag`.

  ### Changed
  - Updated documentation to reflect the latest provider features.
  ```

### Update Examples

1. Navigate to the `/examples` directory.
2. Add or update the example files:
  - For the Low-Code Monitored RAG example, ensure the `main.tf` file includes the latest configuration for the provider.
  - Example `main.tf`:
    ```hcl
    provider "datarobot" {
     api_token = var.datarobot_api_token
    }

    resource "datarobot_project" "example" {
     name = var.use_case_name
    }
    ```
3. For additional examples and detailed usage, refer to the [examples README](https://github.com/datarobot-community/terraform-provider-datarobot/tree/main/examples/README.md).

### Generate Documentation

Documentation is generated based on the `Description` and `MarkdownDescription` fields in the schemas.

1. Run the following command to generate updated documentation:
  ```bash
  make generate
  ```
2. Verify that the generated documentation reflects the latest changes.

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

**It's generally a good idea to run acceptance tests affected by your change manually before submitting a PR.** They can be run individually, setting `TF_ACC=1` and `DATAROBOT_API_TOKEN` to appropriate values. Please ensure the service account associated with the provided `DATAROBOT_API_TOKEN` has the necessary features enabled to run the tests.

Getting the mock calls right for integration tests can be challenging. You may be surprised at how many read calls are
issued at each stage. Some optional logging has been enabled which can help with this. Enable it by setting the env var `TRACE_API_CALLS=1`. If you first run an acceptance test that performs the real operations, ie `TF_ACC=1 TRACE_API_CALLS=1`, you can then use this record to determine the mock calls.

Tests should generally cover all of a resource's CRUD operations, plus `import`. That generally means at least three test steps. Data sources can either have their own simple test method or, if convenient, be dropped into an existing resource test.

There are currently no permanent fixtures for acceptance tests to use. Each test needs to create the resources it depends on.

## Releasing

To create a new release simply make a Release in GitHub. The [Release
Action](https://github.com/datarobot-community/terraform-provider-datarobot/actions/workflows/release.yml)
will pick it up and run go-releaser. After 5 minutes or so, Hashicorp
will pick up the release on the [primary provider
site](https://registry.terraform.io/providers/datarobot-community/datarobot/latest)
side with the latest version, and it is ready to go and get released
to Pulumi via our [Pulumi
Bridge](https://github.com/datarobot-community/pulumi-datarobot). You
can see the instructions for testing and releasing the Pulumi provider
in that repo's documentation.


## Questions?

If you're an external contributor, first of all, thank you! The best way to contact us is with a GitHub issue. Any and all feedback is welcome.
