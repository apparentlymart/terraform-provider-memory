# "Memory" provider for Terraform

This is a provider plugin for Terraform that provides a "memory" utility, which
you can update selectively when needed but leave unchanged otherwise.

This can be useful when using write-only attributes with resource types that
expect you to maintain some sort of "version" that changes whenever the
write-only attribute needs to be updated. You can use the `memory` resource
type to "remember" the version value most recently used and write a new
value only when you intend to make a change.

```hcl
terraform {
  required_providers {
    memory = {
      source = "apparentlymart/memory"
    }
    random = {
      source = "hashicorp/random"
    }
    aws = {
      source = "hashicorp/aws"
    }
  }
}

variable "new_password_version" {
  type    = string
  default = null

  description = "Set this to a value different from the password_version output value to force a new password to be generated and stored."
}

resource "memory" "password_version" {
  # Whenever var.new_password_version has a non-null value,
  # the "value" attribute will be unknown during the planning
  # phase and then updated to the new value in the apply phase.
  #
  # If var.new_password_version is null then the previous
  # value of the "value" attribute is retained.
  new_value = var.new_password_version
}

ephemeral "random_password" "password" {
  # We only need to generate a password when we have a new password version,
  # because otherwise we'll just preserve whatever password was previously
  # stored.
  count = var.new_password_version != null

  length = 32
}

resource "aws_secretsmanager_secret_version" "example" {
  secret_id     = aws_secretsmanager_secret.example.id

  # The following two arguments together ensure that the password gets
  # updated to a new randonly-selected value when var.new_password_version
  # is set, or left unchanged when that variable is null.
  secret_string_wo         = one(ephemeral.random_password.password[*].result)
  secret_string_wo_version = memory.password_version.value
}

# This exposes the most recently used password version so that an operator
# intending to change the password can set var.new_password_version to anything
# except this value.
output "password_version" {
  value = memory.password_version.value
}
```
