module github.com/kasperlewau/redesk

go 1.12

require (
	gioui.org/ui v0.0.0-20190827145909-41ea609d8efd
	github.com/go-redis/redis v6.15.2+incompatible
	golang.org/x/image v0.0.0-20190828090100-23ea20f87cfc
)

replace gioui.org/ui => ../gio/ui
