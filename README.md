# Liqo provider

    Provider for Terraform to perform just few operations (generate and peering command of liqoctl)

## Getting Started
Follow this example steps to test locally the implemented provider.

### Prerequisites
- terraform
- liqoctl
- go

### Installation
1. in ***.terraform.d*** folder (you should have it in home/\<usr\>/) make directory with this command replacing _architecture_ with your architecture (example: linux_arm64 or linux_amd64):

    ``` mkdir -p /plugins/liqo-provider/liqo/test/0.0.1/\<architecture\>/ ```

    my complete path is the following:
        ```home/\<usr\>/.terraform.d/plugins/liqo-provider/liqo/test/0.0.1/linux_arm64/```

2. from root folder repository move into ***/liqo_provider***

3. run command replacing _path_ with the one created in first step:

    ```go build -o <path>/terraform-provider-test ```

4. from root folder repository move into ***/infrastructure***

5. run command:

    ```terraform init ```

    ```terraform apply -target=module.kind -var-file="variables.tfvars"```

    ```terraform apply -var-file="variables.tfvars"```

## Usage
To edit the provider (and rebuild it) don't forget to remove the ***/infastructure/.terraform*** folder and ***.terraform.lock.hcl*** file 
