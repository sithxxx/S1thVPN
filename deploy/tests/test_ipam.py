from ipaddress import IPv4Address
import pytest 
from sibco.ipam import IPAM, IPAMExhauted

def test_allocates_lowest_free():
    ipam = IPAM("10.77.0.0/22")
    assert ipam.server_addres == IPv4Address("10.77.0.1")
    assert ipam.capacity == 1021
    assert ipam.allocate([]) == IPv4Address("10.77.0.2")
    assert ipam.allocate([IPv4Address("10.77.0.2")]) == IPv4Address("10.77.0.3")
    
def test_exhausted():
    ipam = IPAM("10.77.0.0/30")
    with pytest.raises(IPAMExhauted):
        ipam.allocate([IPv4Address("10.77.0.2")])