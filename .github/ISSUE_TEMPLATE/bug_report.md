---
name: Bug Report
about: You're experiencing an issue with this Packer plugin that is different than the documented behavior.
labels: bug
---

When filing a bug, please include the following headings.

Please delete the example text in this template before submitting.

#### Overview of the Issue

A paragraph or two about the issue you're experiencing.

#### Reproduction Steps

Steps to reproduce this issue

### Plugin and Packer version

Packer and its plugins are distinct binaries.

The Packer version can be found in the packer CLI using the version command `packer version`.

Installed plugins can be found by running `packer plugins installed`.
Packer will use the latest version of an installed plugin unless a different version is specified in the `required_plugins` block. Refer to [Specifying Plugin Requirements] (https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer#specifying-plugin-requirements) for more details.

You can also find the version of the plugin using the plugin binary itself.
Find the path to the plugin in the output of a build by setting the environment variable PACKER_LOG=1.

Then invoke that binary with the describe command, for example

```
$ PACKER_LOG=1 packer build template.pkr.hcl
[...]
/home/elbaj/.packer.d/plugins/github.com/hashicorp/docker/packer-plugin-docker_v1.0.11-dev_x5.0_linux_amd64: plugin process exited
```

From this, I have the path to the plugin executed for the build. I can then execute that binary with the describe command to find the version of the plugin. 

```
$ /home/elbaj/.packer.d/plugins/github.com/hashicorp/docker/packer-plugin-docker_v1.0.11-dev_x5.0_linux_amd64 describe
{"version":"1.0.11-dev","sdk_version":"0.5.4-dev","api_version":"x5.0","builders":["-packer-default-plugin-name-"],"post_processors":["import","push","save","tag"],"provisioners":[],"datasources":[],"protocol_version":"v2"}
```

Calling describe on a plugin binary provides the most accurate version information, as plugin binaries can be easily be renamed for testing purposes. 

Issues posted without these versions often slows down responses, and may require more upfront work from maintainers to identify the cause of the issue.

### Simplified Packer Buildfile

Please include a simplified build file that reproduces this error, try and remove extranaeous information from the template.

If the file is longer than a few dozen lines, please include the URL to the
[gist](https://gist.github.com/) of the log or use the [Github detailed
format](https://gist.github.com/ericclemmons/b146fe5da72ca1f706b2ef72a20ac39d)
instead of posting it directly in the issue.

### Operating system and Environment details

OS, Architecture, and any other information you can provide about the environment.

### Log Fragments and crash.log files

Include appropriate log fragments. If the log is longer than a few dozen lines,
please include the URL to the [gist](https://gist.github.com/) of the log or
use the [Github detailed format](https://gist.github.com/ericclemmons/b146fe5da72ca1f706b2ef72a20ac39d) instead of posting it directly in the issue.

Set the env var `PACKER_LOG=1` for maximum log detail.
