# `data.git`

The `git` data provider clones a git repository
to a specific directory  

### Opening the existing repository

```hcl 
data "git" "this" {
  url = "."
}
```

### Cloning a new repository
```hcl
{{#include ../../../examples/git/togomak.hcl}}
```

### Reusing remote stages, using togomak 
```hcl
{{#include ../../../examples/remote-stages/togomak.hcl}} 
```

## Arguments Reference

- [`url`](#url) - The URL of the repository to clone.
- [`tag`](#tag) - The tag to checkout. `tag` takes precedence over `branch`.
- [`ref`](#ref) - The reference to checkout, in the format `refs/tags/v3` for example. `ref` takes precedence over `tag` and `branch`.
- [`branch`](#branch) - The branch to checkout. `ref` and `tag` takes precedence over `branch`.
- [`destination`](#destination) - The destination directory to clone the repository to, defaults to `"memory"`, which clones into a temporary directory managed by `togomak`
- [`commit`](#commit) - The commit to checkout.
- [`depth`](#depth) - The depth of the clone.
- [`ca_bundle`](#ca_bundle) - The path to a CA bundle file or directory, (deprecated, does nothing).
- [`auth`](#auth) - The authentication credentials to use when cloning the repository. Structure [documented below](#auth)
- [`files`](#files) - The files to checkout from the repository. Accepts an array of file paths.

## Attributes Reference 
- [`last_tag`](#last-tag) - The latest tag in the repository, defaults to `""`
- [`commits_since_last_tag`](#commits-since-last-tag) - The number of commits since the last tag, defaults to `0`
- [`sha`](#sha) - The SHA of the commit, defaults to `""`
- [`ref`](#ref) - The ref of the commit, in the format `refs/heads/<branch>` or `refs/tags/<tag>`, defaults to `""`
- [`is_tag`](#is-tag) - Whether the ref is a tag, defaults to `false`
- [`files`](#files) - The files checked out from the repository. Returns a map, with the keys being the file paths and the values being the file contents.
- [`branch`](#branch) - The branch checked out from the repository. Returns a string.
- [`destination`](#destination) - The destination where the repository is stored. 

---
<a id="auth"></a>
The `auth` block supports:
- [`username`](#username) - The username to be used. If password is specified, and if the username is empty, it falls back to `oauth2`
- [`password`](#password) - The password to be used for connecting to the private repository. 




