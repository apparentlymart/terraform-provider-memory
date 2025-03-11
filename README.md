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

variable "update_password" {
  type    = bool
  default = false
}

# This is using random_password just because at the time of writing it's
# the only ephemeral resource type available in the hashicorp/random
# provider, but this is really just a way to get a random value for
# the "version" and will not actually be used as a password.
ephemeral "random_password" "password_version" {
  count = var.update_password ? 1 : 0

  length = 256
}

resource "memory" "password_version" {
  # This awkward expression arranges for the memory to be updated
  # whenever there's an instance of ephemeral.random_password.password_version,
  # and set to a sha256 of the randomly-generated string.
  new_value = one([
    for p in ephemeral.random_password.password_version : nonsensitive(sha256(p.result))
  ])
}

# This one _is_ generating a password, as a placeholder for whatever you need
# to do to get an ephemeral sensitive value to write into the write-only
# attribute that needs updating.
ephemeral "random_password" "password" {
  count = var.update_password ? 1 : 0

  length = 32
}

# Now you can use the memory in combination with the possibly-generated
# random password to write a new password into (for example) AWS secrets
# manager whenever var.update_password is set to true.
resource "aws_secretsmanager_secret_version" "example" {
  secret_id     = aws_secretsmanager_secret.example.id

  secret_string_wo         = one(ephemeral.random_password.password[*].result)
  secret_string_wo_version = memory.password_version.value
}
```

(This is still under development and doesn't quite work yet.)
