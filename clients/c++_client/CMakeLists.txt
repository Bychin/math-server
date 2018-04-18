project ("C++ clients for math-server")
set (CMAKE_CXX_STANDARD 11)

# Post client
add_executable(post_client post_client.cpp json.hpp)
# Calc client
#add_executable(calc_client calc_client.cpp json.hpp)