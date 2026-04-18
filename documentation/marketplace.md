# Building your own Corporate Marketplace

taito is designed with enterprise readiness in mind. By leveraging the **OCI (Open Container Initiative)** standard, taito allows companies to instantly create an internal, private marketplace for AI coding skills, agents, and bundles using their existing infrastructure.

## OCI Packaging and Registries

When you run `taito package`, your skill or agent is bundled into a standard OCI artifact. This means you can distribute taito packages using **any standard OCI container registry**—the exact same registries you already use for Docker images.


To publish a skill internally, a developer simply packages and pushes it:
```bash
$ taito package
$ taito push registry.internal.company.com/ai-tools/my-skill:1.0.0
```

Other developers can then seamlessly install it:
```bash
$ taito install registry.internal.company.com/ai-tools/my-skill:1.0.0
```

---

## Usage in Corporate Environments

Using OCI registries for internal AI tooling distribution aligns with existing infrastructure patterns:

1. `Infrastructure reuse`: Leverages existing Docker/OCI registries. Most companies already have private registries for container images, so they can use the same for taito packages without additional setup.
   
2. `Supply chain`: Uses standard OCI digests (SHA256) to ensure artifact immutability. This can be used for SBOM creation also verifying a specific version of a skill or agent is save to use.

3. `Best practices`: Creating bundles with company-wide best practices, internal APIs, and shared context to ensure consistency across teams.
   


---

## Bundles: Best Practices and Guidelines

Bundles allow grouping multiple skills and agents into a single package. When maintaining internal tools, consider the following patterns:

* `Role-specific bundles`: Create bundles tailored to specific teams (e.g., `frontend-bundle`, `sre-bundle`) rather than requiring users to install tools individually.
  

* `Standardized onboarding`: Maintain a core engineering bundle that provides necessary internal context, API guidelines, and scripts for new developers.
 

* `Version pinning`: Pin included skills to specific semantic versions to maintain stability and prevent upstream changes from breaking workflows.
  

* `Metadata`: Populate the `description`, `keywords`, and `author` fields in `taito.spec` to improve discoverability within the registry UI.

---

## Self-Hostable Marketplace UI

While OCI registries completely solve the *storage and distribution* problem, we know that *discovery* (searching, browsing, and reading documentation for available internal skills) is just as important.

**We also plan to build a self-hostable marketplace frontend** that connects to your internal OCI registry, providing a beautiful, searchable UI for your developers to find the tools they need. 

For more information, check the **roadmap**!
