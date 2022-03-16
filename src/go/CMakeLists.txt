add_custom_target(copy_related_files
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/README.md" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/forcepser.txt"
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/src/lua/_entrypoint.lua" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/_entrypoint.lua"
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/src/setting.txt-template" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/setting.txt-template"
  COMMAND ${CMAKE_COMMAND} -E copy "${PROJECT_SOURCE_DIR}/src/setting.txt-template-old" "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/setting.txt-template-old"
)

find_package(Git REQUIRED)
execute_process(
  COMMAND ${GIT_EXECUTABLE} tag --points-at HEAD
  OUTPUT_VARIABLE _git_tag
  ERROR_QUIET
  OUTPUT_STRIP_TRAILING_WHITESPACE
)
if ("${_git_tag}" STREQUAL "")
  set(_git_tag "vX.X.X")
endif()
message(STATUS "git tag: ${_git_tag}")

execute_process(
  COMMAND ${GIT_EXECUTABLE} rev-parse --short HEAD
  OUTPUT_VARIABLE _git_revision
  ERROR_QUIET
  OUTPUT_STRIP_TRAILING_WHITESPACE
)
if ("${_git_revision}" STREQUAL "")
  set(_git_revision "unknown")
endif()
message(STATUS "git revision: ${_git_revision}")

add_custom_target(forcepser_exe ALL
  COMMAND go build -x
    "-ldflags=\"-s -X \\\"main.version=${_git_tag} ( ${_git_revision} )\\\"\""
    -o "${CMAKE_BINARY_DIR}/${EXPORT_DIR}/forcepser.exe"
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
)
add_dependencies(forcepser_exe copy_related_files)
