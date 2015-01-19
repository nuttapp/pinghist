module.exports = function(grunt) {
    require('jit-grunt')(grunt);

    var config = {
        shell: {
            del_db: {
                command: "rm pinghist.db",
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
                command: "db_info.sh"
            }
        },
        watch: {
            files: ['**/*.go', '**/*.sh'],
            tasks: ['clear', 'shell:del_db', 'shell:go_test', 'shell:db_info']
        }
    };
    grunt.initConfig(config);

    grunt.registerTask('default', ['clear', 'watch']);
};
