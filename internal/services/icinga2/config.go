package icinga2

import (
	"fmt"
	"github.com/icinga/icinga-testing/internal"
	"github.com/icinga/icinga-testing/services"
)

func WriteInitialConfig(i services.Icinga2Base) {
	i.DeleteConfigGlob("etc/icinga2/conf.d/*.conf")

	i.WriteConfig("etc/icinga2/conf.d/icinga-testing-api-user.conf", []byte(fmt.Sprintf(`
		object ApiUser %q {
			password = %q
			permissions = ["*"]
		}
	`, internal.Icinga2DefaultUsername, internal.Icinga2DefaultPassword)))
}
