package backend

import "testing"

var testManifest = `
{
    "acVersion": "0.5.1",
    "acKind": "PodManifest",
    "apps": [
        {
            "name": "reduce-worker",
            "image": {
                "name": "example.com/reduce-worker",
                "id": "sha512-...",
                "labels": [
                    {
                        "name":  "version",
                        "value": "1.0.0"
                    }
                ]
            },
            "app": {
                "exec": [
                    "/bin/reduce-worker",
                    "--debug=true",
                    "--data-dir=/mnt/foo"
                ],
                "group": "0",
                "user": "0",
                "mountPoints": [
                    {
                        "name": "work",
                        "path": "/mnt/foo"
                    }
                ]
            },
            "mounts": [
                {"volume": "work", "mountPoint": "work"}
            ]
        },
        {
            "name": "backup",
            "image": {
                "name": "example.com/worker-backup",
                "id": "sha512-...",
                "labels": [
                    {
                        "name": "version",
                        "value": "1.0.0"
                    }
                ]
            },
            "app": {
                "exec": [
                    "/bin/reduce-backup"
                ],
                "group": "0",
                "user": "0",
                "mountPoints": [
                    {
                        "name": "backup",
                        "path": "/mnt/bar"
                    }
                ],
                "isolators": [
                    {
                        "name": "resource/memory",
                        "value": {"limit": "1G"}
                    }
                ]
            },
            "mounts": [
                {"volume": "work", "mountPoint": "backup"}
            ],
            "annotations": [
                {
                    "name": "foo",
                    "value": "baz"
                }
            ]
        },
        {
            "name": "register",
            "image": {
                "name": "example.com/reduce-worker-register",
                "id": "sha512-...",
                "labels": [
                    {
                        "name": "version",
                        "value": "1.0.0"
                    }
                ]
            }
        }
    ],
    "volumes": [
        {
            "name": "work",
            "kind": "host",
            "source": "/opt/tenant1/work",
            "readOnly": true
        }
    ],
    "isolators": [
        {
            "name": "resource/memory",
            "value": {
                "limit": "4G"
            }
        }
    ],
    "annotations": [
        {
           "name": "ip-address",
           "value": "10.1.2.3"
        }
    ],
    "ports": [
        {
            "name": "ftp",
            "hostPort": 2121
        }
    ]
}`

func TestGetManifest(t *testing.T) {
	var testBackend = NewBackend()

	token, err := testBackend.RegisterPod("abcd-efgh-ijkl-mnop", []byte(testManifest), "")
	if err != nil {
		t.Fatalf("Failed to register pod: %v", err)
	}

	appDef := testBackend.GetAppDefinition(token, "backup")
	if appDef == nil {
		t.Fatal("Failed to obtain app manifest")
	}

}
