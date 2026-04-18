# Quickstart Guide

## Installation

### via NPM
Note: -g is required to install taito globally
```bash
npm install -g @taito-project-eu/taito
```

### Or just via precompiled binaries / packages (deb, rpm, apk)

Just check the [`latest release`](https://github.com/taito-project/taito/releases/latest) for the precompiled binaries and packages.


## Quickstart Guide

- **Installation:** Install `taito` using one of the methods described above.

### Run taito setup command to set up taito for the first time
```bash
taito setup
```
Select the tools you use and taito will automatically set up the configuration for you. You can always run `taito setup` again to change the configuration.


### Install Skills from Github

```bash
taito install github.com/anthropics/skills
```
Use space to select the skills you want to install and enter to confirm. 

### List installed skills
```bash
taito list
```
### Update installed skills
```bash
taito update
```
This will check for updates for all installed skills and update them if there are new versions available.

### Uninstall a skill
```bash
taito uninstall <skill-id-from-taito-list>
```

### Packaging your own skills and agents as OCI artifacts

To start with packaging your own skills and agents, we start with creating a `taito.spec` file. This can be done manually or by using the `taito init` command, which will guide you through the process of creating a `taito.spec` file.

This can be one skill or agent, but it can also be a bundle of skills and agents.

Please check the example `taito.spec` file in the [taito.spec/examples/](taito.spec/examples/) directory for reference.

```bash 
taito init
```
After creating the `taito.spec` file, you can use the `taito package` command to package your skills and agents into OCI artifacts.

```bash
taito package your.oci.registry/your-namespace/your-artifact:tag --spec taito.spec
```

This can then be used for installing and publishing your oci artifact using the `login`, `push`, and `install` commands. Check out our [command documentation](https://taito-project.eu/documentation/commands) for more information on how to use these commands. 
