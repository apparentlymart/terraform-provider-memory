# "memory" provider

This is a utility provider that provides a resource type that can act as a
memory cell that retains a value in the state until it's explicitly overwritten.

This provider requires Terraform v1.11 or later, because it uses both ephemeral
values and write-only attributes.

To use this provider you must declare it as required by each module that will
declare a `memory` resource:

```hcl
terraform {
  required_providers {
    memory = {
      source = "apparentlymart/memory"
    }
  }
}
```
