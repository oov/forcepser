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

add_custom_target(forcepser_exe ALL COMMAND ${CMAKE_COMMAND} -E env GOROOT="$ENV{MINGW_PREFIX}/lib/go"
  go build -x
    "-ldflags=\"-s -X \\\"main.version=${git_tag} ( ${git_revision} )\\\"\""
    -o "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/forcepser.exe"
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  USES_TERMINAL
)
add_dependencies(forcepser_exe generate_readme copy_related_files)
