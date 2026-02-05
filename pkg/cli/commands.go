package cli

// PkgInstall
type PkgInstallArgs struct {
	Package string
}
type PkgInstallFlags struct {
	Force bool
}

// PkgList
type PkgListArgs struct {
	Package string
}
type PkgListFlags struct {
	All   bool
	Index bool
}

// RecipeRepl
type RecipeReplArgs struct {
	File string
}

// CaveUse
type CaveUseArgs struct {
	Cave string
}

// CaveRun
type CaveRunArgs struct {
	Command string
}
type CaveRunFlags struct {
	Variant string
}

// CaveAddPkg
type CaveAddPkgArgs struct {
	Package string
}

// DiskUninstall
type DiskUninstallFlags struct {
	Force bool
}

// RemoteAdd
type RemoteAddArgs struct {
	Name string
	URL  string
}
