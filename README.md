<div align=center>
    <img src=https://raw.githubusercontent.com/taito-project/.github/main/assets/header.png alt="taito project logo">
</div>
<div align=center>
      <img src=https://img.shields.io/badge/go-1.25.8-e59442 alt="goversion">
      <img src=https://img.shields.io/badge/LICENSE-MIT-e59442 alt="license">
      <a href="https://taito-project.eu">
         <img src=https://img.shields.io/badge/WEBSITE-taito--project.eu-e59442 alt="website">
      </a>
      <a href="https://taito-project.eu/documentation/commands">
         <img src=https://img.shields.io/badge/DOCUMENTATION-taito--project.eu-e59442 alt="documentation">
      </a>
</div>

`taito` is a packaging manager for AI skills/agents.


## Why use `taito`?

taito aims to be a standard way of packaging multiple skills and there dependencies into OCI artifacts, so called `bundles`. These bundles can then be easily installed and updated using the `taito` CLI. Make it easy to share and distribute skills for specific use cases like `frontend bundle` or `devops bundle`.

It was designed with corporate environments in mind, where there is a need for a clear overview of what is contained in the skills and how they are shared across the organization.
This is also why taito can detect update to skills bundles stored in OCI registries making it a fully featured package manager for skills.

We don't want to reimplement the wheel, so taito can work with standard OCI registries which are widely used inside organizations, making it easy to adopt. 


![Preview](https://raw.githubusercontent.com/taito-project/.github/main/assets/taito-promo.gif)


For more information, please see the [taito (website)](https://taito-project.eu) or the [taito command documentation (website)](https://taito-project.eu/documentation/commands).

## Installation

#### via install script
```bash
curl -fsSL https://raw.githubusercontent.com/taito-project/taito/main/install.sh | sh
```

#### via Go
```bash
   go install github.com/taito-project/taito
```

#### via NPM
Note: -g is required to install taito globally
```bash
npm install -g @taito-project-eu/taito
```

#### Or just via precompiled binaries / packages (deb, rpm, apk)

Just check the latest release for the precompiled binaries and packages.


## Quick start Guide

- **Installation:** Install `taito` using one of the methods described above.

#### Run taito setup command to set up taito for the first time
```bash
taito setup
```
Select the tools you use and taito will automatically set up the configuration for you. You can always run `taito setup` again to change the configuration.


#### Install Skills from Github

```bash
taito install github.com/anthropics/skills
```
Use space to select the skills you want to install and enter to confirm. 

#### List installed skills
```bash
taito list
```
#### Update installed skills
```bash
taito update
```
This will check for updates for all installed skills and update them if there are new versions available.

#### Uninstall a skill
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


## taito.spec

`taito.spec` is a specification file that defines the metadata and configuration for your skills and agents. It follows the v0.1.0 specification, which includes fields such as `name`, `type`, `description` and more.

The goal is to have a standardized format for defining skills and agents and packaging them into OCI artifacts.


## Development & Contributing

We welcome contributions! Please see [DEVELOPER.md](DEVELOPER.md) for detailed information on the architecture, technical details, and how to get started with development. Or join our community on [Discord server](https://discord.gg/k4TGWxnG) to discuss and contribute to the project.
