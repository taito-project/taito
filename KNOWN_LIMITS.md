These are known issues with the taito:

### Github related issues
Because taito uses plain GitHub api calls we need to work with the ratelimiting of github. This means that if a repository has more than 100 tags, taito will only consider the last 100 tags when checking for updates. This can lead to issues if a skill is updated but the new version is not among the last 100 tags.
