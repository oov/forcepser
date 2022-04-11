cmake_minimum_required(VERSION 3.0)
project(forcepser NONE)

set(EXPORT_DIR "bin" CACHE STRING "export directory")
if ("${EXPORT_DIR}" STREQUAL "")
  set(EXPORT_DIR "bin" CACHE STRING "export directory" FORCE)
endif()

set(ASAS_VERSION "v0.2.2")
set(ASAS_URL "https://github.com/oov/asas/releases/download/${ASAS_VERSION}/asas_${ASAS_VERSION}.zip")
set(ASAS_DIR "${EXPORT_DIR}/asas")
add_custom_target(extract_asas ALL COMMAND
  ${CMAKE_COMMAND}
  -Dlocal_dir="${PROJECT_BINARY_DIR}"
  -Durl="${ASAS_URL}"
  -Ddir="${ASAS_DIR}"
  -P "${CMAKE_CURRENT_SOURCE_DIR}/src/cmake/extract-zip.cmake"
)

add_subdirectory(src/go)
