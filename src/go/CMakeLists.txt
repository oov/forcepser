set(GO_URL "https://go.dev/dl/go1.20.7.windows-amd64.zip")
string(REGEX MATCH "[^/]+$" GO_ARCHIVE_NAME "${GO_URL}")
set(GO_ARCHIVE_PATH "${CMAKE_CURRENT_BINARY_DIR}/${GO_ARCHIVE_NAME}")
string(REGEX REPLACE "\\.[^.]+$" "" GO_ARCHIVE_NOEXT "${GO_ARCHIVE_NAME}")
set(GO_DIR "${CMAKE_CURRENT_BINARY_DIR}/go")
set(GO_EXE "${GO_DIR}/bin/go.exe")
if(NOT EXISTS "${GO_ARCHIVE_PATH}")
  file(DOWNLOAD "${GO_URL}" "${GO_ARCHIVE_PATH}")
endif()
if(NOT EXISTS "${GO_DIR}")
  execute_process(
    COMMAND ${CMAKE_COMMAND} -E tar xzf ${GO_ARCHIVE_PATH}
    WORKING_DIRECTORY "${CMAKE_CURRENT_BINARY_DIR}"
  )
endif()

add_custom_target(${PROJECT_NAME}_format
  COMMAND ${CMAKE_COMMAND} -E env GOROOT="${GO_DIR}" ${GO_EXE} fmt
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  USES_TERMINAL
)

find_package(Git REQUIRED)
execute_process(
  COMMAND ${GIT_EXECUTABLE} tag --points-at HEAD
  OUTPUT_VARIABLE git_tag
  ERROR_QUIET
  OUTPUT_STRIP_TRAILING_WHITESPACE
)
if ("${git_tag}" STREQUAL "")
  set(git_tag "vX.X.X")
endif()

execute_process(
  COMMAND ${GIT_EXECUTABLE} rev-parse --short HEAD
  OUTPUT_VARIABLE git_revision
  ERROR_QUIET
  OUTPUT_STRIP_TRAILING_WHITESPACE
)
if ("${git_revision}" STREQUAL "")
  set(git_revision "unknown")
endif()

add_custom_target(generate_readme COMMAND
  ${CMAKE_COMMAND}
  -Dinput_file="${PROJECT_SOURCE_DIR}/README.md"
  -Doutput_file="${CMAKE_BINARY_DIR}/${EXPORT_DIR}/forcepser.txt"
  -Dnewline="CRLF"
  -Dgit_tag="${git_tag}"
  -Dgit_revision="${git_revision}"
  -P "${PROJECT_SOURCE_DIR}/src/cmake/replace.cmake"
)
add_custom_target(copy_related_files
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/src/lua/_entrypoint.lua" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/_entrypoint.lua"
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/src/setting.txt-template" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/setting.txt-template"
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/src/setting.txt-template-old" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/setting.txt-template-old"
)

add_custom_target(forcepser_exe ALL COMMAND ${CMAKE_COMMAND} -E env GOROOT="${GO_DIR}"
  "${GO_EXE}" build -x
    "-ldflags=\"-s -X \\\"main.version=${git_tag} ( ${git_revision} )\\\"\""
    -o "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/forcepser.exe"
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  USES_TERMINAL
)
add_dependencies(forcepser_exe generate_readme copy_related_files)
