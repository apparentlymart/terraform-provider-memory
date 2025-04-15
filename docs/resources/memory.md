# `memory` resource type

The `memory` resource type acts as a memory cell that "remembers" a value
in the Terraform state until it's overwritten.

```hcl
terraform {
  required_providers {
    memory = {
      source = "apparentlymart/memory"
    }
  }
}

# Typical use of this resource type requires a value that is non-null
# whenever the memory should be written and null when the memory should
# retain its previous value.
variable "new_value" {
  type    = string
  default = null
}

resource "memory" "example" {
  new_value = var.new_value
}

output "value" {
  value = memory.example.value
}
```

## Arguments

- `new_value`: The new value to write to the memory cell, or `null` to retain
  the previous value.

  This argument is required when a new `memory` instance is being created, to
  define the initial value of the memory.

## Attributes

- `value`: The current value stored in the memory.

  If the `new_value` attribute is set during the planning phase then the
  `value` attribute is unknown to represent that its value will be updated
  during the apply phase. The final value is equal to the `new_value`
  argument.

  If `new_value` is null during the planning phase then `value` is the
  previously-stored value.

## Practical Example

The primary purpose of this resource type is to support the use of write-only
attributes on other resource types that require you to set a "version" argument
whose value changes each time the corresponding write-only argument needs to
be updated.

`memory` can be used to remember the previously-stored "version" value so that
you don't need to constantly resupply the same value on subsequent plan/apply
rounds. Instead, you can use an input variable that is left completely unset
unless the write-only value needs to change.

For example, using the `aws_secretsmanager_secret_version` resource type from
the `hashicorp/aws` provider to regenerate a password only when a new
password version is provided:

```hcl
variable "new_password_version" {
  type    = number
  default = null
}

resource "memory" "example" {
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
# intending to change the password can set var.new_password_version to something
# different than this value.
output "password_version" {
  value = memory.password_version.value
}
```

During initial creation of this configuration, `new_password_version` **must**
be set to an arbitrary initial value.

On subsequent plan/apply rounds, leaving `new_password_version` unset will
leave the stored password unchanged, without the operator needing any knowledge
of the old password. Setting `new_password_version` to a non-null value will
update the value stored in the memory and simultaneously trigger the
`aws_secretsmanager_secret_version.example` object to be replaced, thereby
storing a newly-generated password in secrets manager without retaining that
password in the Terraform state.

The `password_version` output value exposes the most recently used password
version so that an operator intending to reset the password can choose a
different value to use.
