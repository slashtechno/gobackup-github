package internal

import "github.com/spf13/viper"

// Creating a Viper object as it makes it easier to migrate to multiple Vipers if needed
var Viper = viper.New()
