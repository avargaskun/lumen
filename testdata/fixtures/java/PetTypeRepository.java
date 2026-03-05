package org.springframework.samples.petclinic.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.samples.petclinic.model.PetType;
import java.util.List;

public interface PetTypeRepository extends JpaRepository<PetType, Integer> {
    List<PetType> findAll();
    PetType findByName(String name);
}
