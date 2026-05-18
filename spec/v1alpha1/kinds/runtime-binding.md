# RuntimeBinding

`RuntimeBinding` is optional.

It describes where policy enforcement may happen later:

```text
generic webhook
gateway plugin
local sidecar
MCP wrapper
native gateway policy
```

Generation mode must work without a runtime binding.
