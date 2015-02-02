module.exports = function(grunt) {
    require('jit-grunt')(grunt);

    var config = {
        shell: {
            go_install: {
                command: "go install ./...",
            },
            del_db: {
                command: "rm dal/pinghist.db",
                options: {
                    failOnError: false,
                    stdout: false,
                    stderr: false
                }
            },
            go_test: {
                command: "test.sh",
                options: {
                    execOptions: {
                        maxBuffer: 4000*1024 
                    }
                }
            },
            db_info: {
                command: "test_db_info.sh"
            }
        },
        watch: {
            files: ['**/*.go', '**/*.sh', '!**/node_modules/**', '!**/.git/**'],
            tasks: ['run_test']
        }
    };
    grunt.initConfig(config);

    grunt.registerTask('default', ['clear', 'watch']);
    grunt.registerTask('run_test', ['clear', 'shell:go_install', 'shell:go_test']);

};
