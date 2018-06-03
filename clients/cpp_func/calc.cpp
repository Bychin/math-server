#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include "json.hpp"

using json = nlohmann::json;

//const std::string dataCalcPath = "./dataCalc.txt";

// splits string in vector ("1,2,3" -> {1, 2, 3})
std::vector<int> splitToInt(std::string target, char delim) {
    std::istringstream ss(target);
	std::string token;
	std::vector<int> v;

	while(std::getline(ss, token, delim)) {
		v.push_back(std::stoi(token));
	}
    return v;
}

int main(int argc, char **argv) {
    std::cout << "Enter vector components:\n";
    std::string line;
    getline(std::cin, line);

    int scalar;
    std::cout << "Enter scalar:\n";
    std::cin >> scalar;

    json j;
    j["vector"] = splitToInt(line, ',');
    j["scalar"] = scalar;

    const std::string dataCalcPath = argv[1];
    std::ofstream myfile(dataCalcPath);
    if (myfile.is_open()) {
        myfile << j;
        myfile.close();
    }
    return 0;
}
