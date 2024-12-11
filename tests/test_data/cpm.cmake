project('fds')
CPMAddPackage("gh:catchorg/Catch2@2.5.0")
CPMAddPackage("gh:ericniebler/range-v3#0.11.0")
CPMAddPackage("gh:jbeder/yaml-cpp#yaml-cpp-0.6.3@0.6.3")    
CPMAddPackage(
  NAME nlohmann_json
  VERSION 3.9.1
  GITHUB_REPOSITORY nlohmann/json
  OPTIONS 
    "JSON_BuildTests OFF"
)
# using `CPM_SOURCE_CACHE` is strongly recommended
CPMAddPackage(
  NAME Boost
  VERSION 1.77.0
  GITHUB_REPOSITORY "boostorg/boost"
  GIT_TAG "boost-1.77.0"
)
# the install option has to be explicitly set to allow installation
CPMAddPackage(
  GITHUB_REPOSITORY jarro2783/cxxopts
  VERSION 2.2.1
  OPTIONS "CXXOPTS_BUILD_EXAMPLES NO" "CXXOPTS_BUILD_TESTS NO" "CXXOPTS_ENABLE_INSTALL YES"
)
CPMAddPackage(
  NAME lua
  GIT_REPOSITORY https://github.com/lua/lua.git
  VERSION 5.3.5
  DOWNLOAD_ONLY YES
)

if (lua_ADDED)
  # lua has no CMake support, so we create our own target

  FILE(GLOB lua_sources ${lua_SOURCE_DIR}/*.c)
  list(REMOVE_ITEM lua_sources "${lua_SOURCE_DIR}/lua.c" "${lua_SOURCE_DIR}/luac.c")
  add_library(lua STATIC ${lua_sources})

  target_include_directories(lua
    PUBLIC
      $<BUILD_INTERFACE:${lua_SOURCE_DIR}>
  )
endif()
CPMAddPackage("gh:catchorg/Catch2@2.5.0")