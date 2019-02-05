# Bakery
Bake your perfect setup - inspired by teraform and chef

Example configuration:

```
/*
 * This is an example config.yum file for bakery.
 */

// Accept the Xcode license first so tools
// like git work without interuption
shell "Accept Xcode License" {
  script = <<EOT
xcodebuild -license accept
EOT
}

// Download and install the Alfred 3 dmg image
dmg "Alfred 3" {
  source = "https://cachefly.alfredapp.com/Alfred_3.8_959.dmg"
  checksum = "1dd15f3063913c22a53eea07f8ffb9b02a61d691416df21f61a57537461da4d5"
}

// Download and install the Google Chrome dmg image
dmg "Google Chrome" {
  source = "https://dl.google.com/chrome/mac/stable/CHFA/googlechrome.dmg"
  checksum = "7f9ae76a661f7b9d40f7e46d5f846d60deefecedebcd5ddc34fba1b05ee2fc6c"
}

// Install dash from their website (.app bundle within the zip)
zip "Dash" {
  source = "https://sanfrancisco.kapeli.com/downloads/v4/Dash.zip"
  checksum = "802c5a63ac72c94ae4c6481529f11795c35f316d3607c9946f777f447b670c50"
  destination = "/Applications/"
}

git "dotfiles" {
  source = "https://github.com/mikemackintosh/dotfiles"
  destination = "~/.dotfiles_test"
  branch = "master"
  depends_on = "Accept Xcode License"
}
```