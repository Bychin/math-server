PROJECT ("C++ clients for math-server")
SET (CMAKE_CXX_STANDARD 11)

# Src declaration
SET(SRC_POST_CLIENT src/post_client.cpp)
SET(SRC_CALC_CLIENT src/calc_client.cpp)
SET(SRC_COMMON src/common.h src/common.cpp)
SET(SRC_JSON src/json.hpp)

# Add executables
add_executable(
	post_client
	${SRC_POST_CLIENT}
	${SRC_COMMON}
	${SRC_JSON}
)

add_executable(
	calc_client
	${SRC_CALC_CLIENT}
	${SRC_COMMON}
	${SRC_JSON}
)