-- Project configuration
set_xmakever("2.5.6")

option("build_mapeditor", { default = true, showmenu = true, description = "Should the map editor be compiled as part of the project? (requires Qt)" })

set_policy("package.requires_lock", true)
add_repositories("burgwar-repo xmake-repo")

set_project("BurgWar")
set_version("0.2.0")

add_requires("cxxopts", "concurrentqueue", "hopscotch-map", "nlohmann_json", "tl_expected", "tl_function_ref")
add_requires("fmt", { configs = { header_only = false, pic = true } })
add_requires("libcurl", { optional = true })
add_requires("nazaraengine 2021.08.28", { alias = "nazara" })
add_requires("nazaraengine~server 2021.08.28", { alias = "nazaraserver", configs = { server = true } })
add_requires("sol2 v3.2.1", { verify = false, configs = { includes_lua = false } })

if is_plat("windows") then
	add_requires("stackwalker master")
end

add_requireconfs("fmt", "stackwalker", { debug = is_mode("debug", "asan") })
add_requireconfs("libcurl", "nazaraengine", "nazaraengine~server", { configs = { debug = is_mode("debug", "asan"), shared = true } })

set_allowedmodes("asan", "debug", "releasedbg")
set_allowedplats("windows", "mingw", "linux", "macosx")
set_allowedarchs("windows|x64", "mingw|x86_64", "linux|x86_64", "macosx|x86_64")
set_defaultmode("debug")

add_rules("mode.asan", "mode.debug", "mode.releasedbg")
add_rules("plugin.vsxmake.autoupdate")

add_headerfiles("thirdparty/natvis/**.natvis")
add_includedirs("include", "src")

add_rpathdirs("@executable_path")

set_languages("c89", "cxx17")
set_rundir("./bin/$(plat)_$(arch)_$(mode)")
set_symbols("debug", "hidden")
set_targetdir("./bin/$(plat)_$(arch)_$(mode)")
set_warnings("allextra")

if is_mode("releasedbg") then
	set_fpmodels("fast")
	add_vectorexts("sse", "sse2", "sse3", "ssse3")
elseif is_mode("asan") then
	set_optimize("none")
end

if is_plat("windows") then
	set_runtimes(is_mode("releasedbg") and "MD" or "MDd")

	add_cxxflags("/bigobj", "/Zc:__cplusplus", "/Zc:referenceBinding", "/Zc:throwingNew")
	add_cxflags("/w44062") -- Enable warning: switch case not handled
	add_cxflags("/wd4251") -- Disable warning: class needs to have dll-interface to be used by clients of class blah blah blah
elseif is_plat("linux") then
	add_cxflags("-fPIC")
	add_syslinks("pthread")
elseif is_plat("mingw") then
	add_cxflags("-Og", "-Wa,-mbig-obj")
	add_ldflags("-Wa,-mbig-obj")
end

target("lua")
	set_group("3rdparties")

	on_load(function (target)
		if (target:is_plat("windows", "mingw")) then
			local static = target:opt("corelib_static") and target:opt("clientlib_static")
			if (not static) then
				target:add("defines", "LUA_BUILD_AS_DLL", { public = true })
			end
		end

		target:set("kind", static and "static" or "shared")
	end)

	set_warnings("none")

	add_options("clientlib_static")
	add_options("corelib_static")
	add_rules("install_bin", "install_symbolfile")

	add_includedirs("contrib/lua/include", { public = true })
	add_headerfiles("contrib/lua/include/**.h")
	add_files("contrib/lua/src/**.c")
	set_symbols("debug")

target("CoreLib")
	set_group("Common")
	set_basename("BurgCore")

	after_load(function (target)
		target:set("kind", target:opt("corelib_static") and "static" or "shared")
	end)

	add_defines("BURGWAR_CORELIB_BUILD")
	add_options("corelib_static")
	add_rules("install_bin", "install_symbolfile")

	add_deps("lua")
	add_headerfiles("include/(CoreLib/**.hpp)", "include/(CoreLib/**.inl)")
	add_headerfiles("src/CoreLib/**.hpp", "src/CoreLib/**.inl")
	add_files("src/CoreLib/**.cpp")
	add_packages("concurrentqueue", "fmt", "hopscotch-map", "nlohmann_json", "sol2", "tl_expected", { public = true })
	add_packages("nazaraserver")
	add_packages("libcurl", { public = true, links = {} })

if is_plat("windows") then
	add_packages("stackwalker")
end

	before_build(function (target)
		local host = os.host()
		local subhost = os.subhost()

		local system
		if (host ~= subhost) then
			system = host .. "/" .. subhost
		else
			system = host
		end

		local branch = "unknown-branch"
		local commitHash = "unknown-commit"
		try
		{
			function ()
				import("detect.tools.find_git")
				local git = find_git()
				if (git) then
					branch = os.iorunv(git, {"rev-parse", "--abbrev-ref", "HEAD"}):trim()
					commitHash = os.iorunv(git, {"describe", "--tags", "--long"}):trim()
				else
					error("git not found")
				end
			end,

			catch
			{
				function (err)
					print(string.format("Failed to retrieve git data: %s", err))
				end
			}
		}

		import("core.project.depend")
		import("core.project.project")
		local tmpfile = path.join(os.projectdir(), "project.autoversion")
		local dependfile = tmpfile .. ".d"
		depend.on_changed(function ()
			print("regenerating version data info...")
			io.writefile("src/CoreLib/VersionData.hpp", string.format([[
const char* BuildSystem = "%s";
const char* BuildBranch = "%s";
const char* BuildCommit = "%s";
const char* BuildDate = "%s";
]], system, branch, commitHash, os.date("%Y-%m-%d %H:%M:%S")))
		end, 
		{
			dependfile = dependfile, 
			files = project.allfiles(), 
			values = {system, branch, commitHash}
		})
	end)

target("ClientLib")
	set_group("Common")
	set_basename("BurgClient")

	after_load(function (target)
		target:set("kind", target:dep("clientlib_static") and "static" or "shared")
	end)

	add_defines("BURGWAR_CLIENTLIB_BUILD")
	add_options("clientlib_static")
	add_rules("install_bin", "install_symbolfile")

	add_deps("CoreLib")
	add_headerfiles("include/(ClientLib/**.hpp)", "include/(ClientLib/**.inl)")
	add_headerfiles("src/ClientLib/**.hpp", "src/ClientLib/**.inl")
	add_files("src/ClientLib/**.cpp")
	add_packages("nazara", { public = true })

target("Main")
	set_group("Common")
	set_basename("BurgMain")

	set_kind("static")
	add_rules("install_symbolfile")

	add_deps("CoreLib")
	add_headerfiles("include/(Main/**.hpp)", "include/(Main/**.inl)")
	add_headerfiles("src/Main/**.hpp", "src/Main/**.inl")
	add_files("src/Main/**.cpp")
	add_packages("nazaraserver")

target("BurgWar")
	set_group("Executable")

	set_kind("binary")
	add_rules("install_symbolfile", "install_metadata")

	add_deps("Main", "ClientLib", "CoreLib")
	add_headerfiles("src/Client/**.hpp", "src/Client/**.inl")
	add_files("src/Client/**.cpp")
	add_packages("nazara")

	if is_plat("windows", "mingw") then
		add_files("src/Client/resources.rc")
	end

	after_install(function (target)
		os.vcp("clientconfig.lua", path.join(target:installdir(), "bin"))
	end)

target("BurgWarServer")
	set_group("Executable")

	set_kind("binary")
	add_rules("install_symbolfile", "install_metadata")

	add_defines("NDK_SERVER")

	add_deps("Main", "CoreLib")
	add_headerfiles("src/Server/**.hpp", "src/Server/**.inl")
	add_files("src/Server/**.cpp")
	add_packages("nazaraserver")

	after_install(function (target)
		os.vcp("serverconfig.lua", path.join(target:installdir(), "bin"))
	end)

target("BurgWarMapTool")
	set_group("Executable")
	set_basename("maptool")

	set_kind("binary")
	add_rules("install_symbolfile")

	add_deps("Main", "CoreLib")
	add_headerfiles("src/MapTool/**.hpp", "src/MapTool/**.inl")
	add_files("src/MapTool/**.cpp")
	add_packages("cxxopts", "nazaraserver")

if has_config("build_mapeditor") then
	target("BurgWarMapEditor")
		set_group("Executable")

		set_kind("binary")
		add_rules("qt.console", "qt.moc")
		add_rules("install_symbolfile", "install_metadata")

		-- Prevents symbol finding issues between Qt5 compiled with C++ >= 14 and Qt5 compiled with C++11
		-- see https://stackoverflow.com/questions/53022608/application-crashes-with-symbol-zdlpvm-version-qt-5-not-defined-in-file-libqt
		if not is_plat("windows") then
			add_cxxflags("-fno-sized-deallocation")
		end

		if is_plat("windows", "mingw") then
			add_files("src/Client/resources.rc")
		end

		add_frameworks("QtCore", "QtGui", "QtWidgets")
		add_deps("Main", "ClientLib", "CoreLib")
		add_headerfiles("src/MapEditor/**.hpp", "src/MapEditor/**.inl")
		add_files("src/MapEditor/Widgets/**.hpp", "src/MapEditor/**.cpp")
		add_packages("nazara")

		on_load(function (target)
			import("detect.sdks.find_qt")

			local qt = find_qt()
			if (not qt) then
				-- Disable building by default if Qt is not found
				target:set("default", false)
			end
		end)

		after_install(function (target)
			os.vcp("editorconfig.lua", path.join(target:installdir(), "bin"))
		end)
end

-- Tasks and options

-- Generates a hash key made of packages confs/version, for CI
task("dephash")
	on_run(function ()
		import("core.project.project")
		import("private.action.require.impl.package")

		local requires, requires_extra = project.requires_str()

		local key = {}
		for _, instance in irpairs(package.load_packages(requires, {requires_extra = requires_extra})) do
			table.insert(key, instance:name() .. "-" .. instance:version_str() .. "-" .. instance:buildhash())
		end

		table.sort(key)

		key = table.concat(key, ",")
		print(hash.uuid4(key):gsub('-', ''):lower())
	end)

	set_menu {
		usage = "xmake dephash",
		description = "Outputs a hash key of current dependencies version/configuration"
	}

-- Copies the resulting target file to the install bin folder (used for .so which are installed to lib/)
rule("install_bin")
	after_install("linux", function(target)
		local binarydir = path.join(target:installdir(), "bin")
		os.mkdir(binarydir)
		os.vcp(target:targetfile(), binarydir)
	end)

-- Copies maps/scripts folders to the install bin folder
rule("install_metadata")
	local metadataInstalled = false
	after_install(function(target)
		if (not metadataInstalled) then
			os.vcp("maps", path.join(target:installdir(), "bin"))
			os.vcp("scripts", path.join(target:installdir(), "bin"))
			metadataInstalled = true
		end
	end)

-- Copies symbol files (such as .pdb) to the install bin folder (required for crashdumps)
rule("install_symbolfile")
	after_install(function(target)
		local symbolfile = target:symbolfile()
		if os.isfile(symbolfile) then
			os.vcp(symbolfile, path.join(target:installdir(), "bin"))
		end
	end)

-- Should the CoreLib be compiled statically? (takes more space)
option("corelib_static")
	set_default(false)
	set_showmenu(true)
	add_defines("BURGWAR_CORELIB_STATIC")

-- Should the ClientLib be compiled statically? (takes more space)
option("clientlib_static")
	set_default(false)
	set_showmenu(true)
	add_defines("BURGWAR_CLIENTLIB_STATIC")