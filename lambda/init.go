package lambda

import "myaws/config"

var settings *config.Settings

func init() {
	settings = config.GetSettings()
}
