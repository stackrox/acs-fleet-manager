# Compliance Operator

Compliance Operator is used to run compliance checks, e.g. NIST or CIS we use it in ACSCS
for testing purposes in our dogfooding instances.
Starting at version 1.5.0 the operator is upgraded automatically.
If the operator breaks it can easily be uninstalled / paused by disabling the flag without production impact.

Value to disable the operator:

```
complianceOperator:
  enabled: false
```
