cmake_minimum_required(VERSION 3.0)
project(forcepser NONE)

set(EXPORT_DIR "bin" CACHE STRING "export directory")
if ("${EXPORT_DIR}" STREQUAL "")
  set(EXPORT_DIR "bin" CACHE STRING "export directory" FORCE)
endif()

add_custom_target(extract_asas ALL COMMAND
  ${CMAKE_COMMAND}
  -Dlocal_dir="${PROJECT_BINARY_DIR}"
  -Durl="https://github.com/oov/asas/releases/download/v0.2.0/asas_v0.2.0.zip"
  -Ddir="${EXPORT_DIR}/asas"
  -P "${CMAKE_CURRENT_SOURCE_DIR}/src/cmake/extract-zip.cmake"
)

add_subdirectory(src/go)
