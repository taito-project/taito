# the art of taito


`taito` is a packaging manager for AI skills/agents.

Using `taito` is the easiest way of installing skills and keeping them up to date. 
It also allows for packaging skills and agents into `OCI artifacts` for bundling and distribution.
As a fallback taito can also work with GitHub repositories that either include a taito.spec file or comply with `skills/` and `agents/` directory structure.

`bundles` bundles are a core concept in taito, they are a collection of skills and agents that are packaged together. Making installing and updating them easier, see `packaging` below for more information.


## Why taito?

`taito` stands for the Norwegian word for "knowledge".
Together with the taito.spec we push for a standardized way of describing skills and agents for installation and distribution.

One thing the industry has taught us over the last decade is that it is always a good idea to know what you ship and what it contains. With taito, we do not want to reinvent the wheel. Many companies already have everything set up for handling OCI artifacts, so we want to leverage that for skills and agents as well.

With taito, you can package your skills and agents into OCI artifacts and use taito to share AI skills and agents across teams and organizations in a safe and standardized way.

Because simply copying files around is not a good solution, taito also supports versioning and updating when something changes in a skill or agent.