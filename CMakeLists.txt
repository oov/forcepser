cmake_minimum_required(VERSION 3.20)
project(forcepser NONE)

set(EXPORT_DIR "bin" CACHE STRING "export directory")
if ("${EXPORT_DIR}" STREQUAL "")
  set(EXPORT_DIR "bin" CACHE STRING "export directory" FORCE)
endif()
set(EXPORT_PATH "${CMAKE_BINARY_DIR}/${EXPORT_DIR}")
if(NOT EXISTS "${EXPORT_PATH}")
  file(MAKE_DIRECTORY "${EXPORT_PATH}")
endif()

set(ASAS_VERSION "v0.2.5")
set(ASAS_URL "https://github.com/oov/asas/releases/download/${ASAS_VERSION}/asas_${ASAS_VERSION}.zip")
string(REGEX MATCH "[^/]+$" ASAS_ARCHIVE_NAME "${ASAS_URL}")
set(ASAS_ARCHIVE_PATH "${CMAKE_CURRENT_BINARY_DIR}/${ASAS_ARCHIVE_NAME}")
if(NOT EXISTS "${ASAS_ARCHIVE_PATH}")
  file(DOWNLOAD "${ASAS_URL}" "${ASAS_ARCHIVE_PATH}")
endif()
set(ASAS_PATH "${EXPORT_PATH}/asas")
if(NOT EXISTS "${ASAS_PATH}")
  file(MAKE_DIRECTORY "${ASAS_PATH}")
  execute_process(
    COMMAND ${CMAKE_COMMAND} -E tar xf ${ASAS_ARCHIVE_PATH}
    WORKING_DIRECTORY "${ASAS_PATH}"
  )
endif()

add_subdirectory(src/go)
